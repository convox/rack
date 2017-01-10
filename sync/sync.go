package sync

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/convox/rack/changes"
	"github.com/fsouza/go-dockerclient"
)

type Stream chan string

type Sync struct {
	Container string
	Local     string
	Remote    string

	docker   *docker.Client
	lock     sync.Mutex
	incoming chan changes.Change
	outgoing chan changes.Change

	incomingBlocks map[string]int
	outgoingBlocks map[string]int
}

type execState struct {
	Running  bool
	Error    error
	ExitCode int
}

func NewSync(container, local, remote string) (*Sync, error) {
	l, err := filepath.Abs(local)

	if err != nil {
		return nil, err
	}

	sync := &Sync{
		Container: container,
		Local:     l,
		Remote:    remote,
	}

	sync.docker, _ = docker.NewClientFromEnv()
	sync.incoming = make(chan changes.Change, 1000)
	sync.outgoing = make(chan changes.Change, 1000)
	sync.incomingBlocks = make(map[string]int)
	sync.outgoingBlocks = make(map[string]int)

	return sync, nil
}

func (s *Sync) Contains(t Sync) bool {
	if !filepath.HasPrefix(t.Local, s.Local) {
		return false
	}

	lr, err := filepath.Rel(s.Local, t.Local)

	if err != nil {
		return false
	}

	rr, err := filepath.Rel(s.Remote, t.Remote)

	if err != nil {
		return false
	}

	return lr == rr
}

func (s *Sync) Start(st Stream) error {
	s.waitForContainer()

	if !filepath.IsAbs(s.Remote) {
		wdb, err := Docker("inspect", "--format", "'{{.Config.WorkingDir}}'", s.Container).Output()
		if err != nil {
			return err
		}

		swdb := string(wdb)
		swdb = strings.TrimSpace(swdb)
		swdb = strings.TrimPrefix(swdb, "'")
		swdb = strings.TrimSuffix(swdb, "'")

		s.Remote = filepath.Join(swdb, s.Remote)
	}

	go s.watchIncoming(st)
	go s.watchOutgoing(st)

	incoming := []changes.Change{}
	outgoing := []changes.Change{}

	tick := time.Tick(1 * time.Second)

	for {
		select {
		case c := <-s.incoming:
			incoming = append(incoming, c)
		case c := <-s.outgoing:
			outgoing = append(outgoing, c)
		case <-tick:
			if len(incoming) > 0 {
				a, r := changes.Partition(incoming)
				s.syncIncomingAdds(a, st)
				s.syncIncomingRemoves(r, st)
				incoming = []changes.Change{}
			}
			if len(outgoing) > 0 {
				a, r := changes.Partition(outgoing)
				s.syncOutgoingAdds(a, st)
				s.syncOutgoingRemoves(r, st)
				outgoing = []changes.Change{}
			}
		}
	}

	return nil
}

func (s *Sync) syncIncomingAdds(adds []changes.Change, st Stream) {
	if len(adds) == 0 {
		return
	}

	tar := []string{"tar", "czf", "-"}

	// docker exec can fail if the argument list is too long
	// limit to 2000 files per exec
	for i := 0; i < len(adds); i += 2000 {

		max := i + 2000
		if max > len(adds) {
			max = len(adds)
		}

		cmd := tar
		batch := adds[i:max]
		for _, a := range batch {
			cmd = append(cmd, filepath.Join(s.Remote, a.Path))
		}

		if err := s.execTar(cmd, st); err == nil {
			st <- fmt.Sprintf("%d files downloaded", len(batch))

		} else {
			st <- fmt.Sprintf("error: %s", err)
			return
		}
	}

	if os.Getenv("CONVOX_DEBUG") != "" {
		for _, a := range adds {
			st <- fmt.Sprintf("<- %s", filepath.Join(a.Base, a.Path))
		}
	}
}

func (s *Sync) execTar(cmd []string, st Stream) error {
	retries := 0
	success := false

	for {
		exec, err := s.docker.CreateExec(docker.CreateExecOptions{
			AttachStdout: true,
			Container:    s.Container,
			Cmd:          cmd,
		})
		if err != nil {
			return err
		}

		r, w := io.Pipe()

		cw, err := s.docker.StartExecNonBlocking(exec.ID, docker.StartExecOptions{
			OutputStream: w,
		})
		if err != nil {
			if cw != nil {
				cw.Close()
			}
			return err
		}

		done := make(chan struct{})
		state := make(chan execState)
		wait := make(chan error)

		go func() {
			wait <- cw.Wait()
		}()
		go tgzReader(s, r, st)
		go inspectExec(exec.ID, s, state, done)

		select {
		case err := <-wait:
			if err != nil {
				return err
			}

			close(done)
			success = true

		case es := <-state:
			cw.Close()

			if retries < 3 {
				retries++
				if es.Error == nil && es.ExitCode == 0 {
					success = true
				}
			} else {
				if es.Error != nil {
					return es.Error
				}
				return fmt.Errorf("failed to sync after retries")
			}
		}

		if success {
			break
		}
	}
	return nil
}

func (s *Sync) syncIncomingRemoves(removes []changes.Change, st Stream) {
	// do not sync removes out from the container for safety
}

func (s *Sync) syncOutgoingAdds(adds []changes.Change, st Stream) {
	if len(adds) == 0 {
		return
	}

	var buf bytes.Buffer

	tgz := tar.NewWriter(&buf)

	for _, a := range adds {
		local := filepath.Join(a.Base, a.Path)

		info, err := os.Stat(local)
		if err != nil {
			continue
		}

		remote := filepath.Join(s.Remote, a.Path)

		s.lock.Lock()
		s.incomingBlocks[a.Path]++
		s.lock.Unlock()

		tgz.WriteHeader(&tar.Header{
			Name:    remote,
			Mode:    0644,
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})

		fd, err := os.Open(local)

		if err != nil {
			st <- fmt.Sprintf("error: %s", err)
			continue
		}

		io.Copy(tgz, fd)
		fd.Close()
	}

	st <- fmt.Sprintf("%d files uploaded", len(adds))

	if os.Getenv("CONVOX_DEBUG") != "" {
		for _, a := range adds {
			st <- fmt.Sprintf("-> %s", filepath.Join(a.Base, a.Path))
		}
	}

	tgz.Close()

	err := s.docker.UploadToContainer(s.Container, docker.UploadToContainerOptions{
		InputStream: &buf,
		Path:        "/",
	})

	if err != nil {
		st <- fmt.Sprintf("error: %s", err)
	}
}

func (s *Sync) syncOutgoingRemoves(removes []changes.Change, st Stream) {
	if len(removes) == 0 {
		return
	}

	cmd := []string{"rm", "-f"}

	for _, r := range removes {
		cmd = append(cmd, filepath.Join(s.Remote, r.Path))
	}

	res, err := s.docker.CreateExec(docker.CreateExecOptions{
		Container: s.Container,
		Cmd:       cmd,
	})

	if err != nil {
		st <- fmt.Sprintf("error: %s", err)
		return
	}

	err = s.docker.StartExec(res.ID, docker.StartExecOptions{
		Detach: true,
	})

	if err != nil {
		st <- fmt.Sprintf("error: %s", err)
		return
	}

	st <- fmt.Sprintf("%d files removed", len(removes))
}

func (s *Sync) uploadChangesDaemon(st Stream) {
	var buf bytes.Buffer

	tgz := tar.NewWriter(&buf)

	data, err := Asset("changed")

	if err != nil {
		st <- fmt.Sprintf("error: %s", err)
	}

	tgz.WriteHeader(&tar.Header{
		Name: "changed",
		Mode: 0755,
		Size: int64(len(data)),
	})

	tgz.Write(data)

	tgz.Close()

	err = s.docker.UploadToContainer(s.Container, docker.UploadToContainerOptions{
		InputStream: &buf,
		Path:        "/",
	})

	if err != nil {
		st <- fmt.Sprintf("error: %s", err)
	}
}

func (s *Sync) waitForContainer() {
	for {
		if res, err := s.docker.InspectContainer(s.Container); err == nil && res.State.Running {
			return
		}
		time.Sleep(1 * time.Second)
	}
}

func (s *Sync) watchIncoming(st Stream) {
	s.uploadChangesDaemon(st)

	res, err := s.docker.CreateExec(docker.CreateExecOptions{
		AttachStdout: true,
		Container:    s.Container,
		Cmd:          []string{"/changed", s.Remote},
	})

	if err != nil {
		st <- fmt.Sprintf("error: %s", err)
		return
	}

	r, w := io.Pipe()

	go func() {
		scanner := bufio.NewScanner(r)

		for scanner.Scan() {
			text := scanner.Text()
			parts := strings.SplitN(text, "|", 3)

			if len(parts) != 3 {
				continue
			}

			// skip incoming removes for now. they make sync hard and not sure we want
			// the container deleting local files anyway
			if parts[0] == "remove" {
				continue
			}

			s.lock.Lock()
			if s.incomingBlocks[parts[2]] > 0 {
				s.incomingBlocks[parts[2]]--
				s.lock.Unlock()
			} else {
				s.lock.Unlock()
				s.incoming <- changes.Change{
					Operation: parts[0],
					Base:      parts[1],
					Path:      parts[2],
				}
			}

		}

		if err := scanner.Err(); err != nil {
			st <- fmt.Sprintf("error: ", err)
		}
	}()

	err = s.docker.StartExec(res.ID, docker.StartExecOptions{
		OutputStream: w,
	})
}

func (s *Sync) watchOutgoing(st Stream) {
	ch := make(chan changes.Change, 1)

	go func() {
		if err := changes.Watch(s.Local, ch); err != nil {
			st <- fmt.Sprintf("error: %s", err)
		}
	}()

	for c := range ch {
		s.lock.Lock()
		if s.outgoingBlocks[c.Path] > 0 {
			s.outgoingBlocks[c.Path]--
			s.lock.Unlock()
		} else {
			s.lock.Unlock()
			s.outgoing <- c
		}
	}
}

func inspectExec(id string, s *Sync, state chan execState, done chan struct{}) {

	es := execState{}

	select {
	case <-time.After(5 * time.Second):
		i, err := s.docker.InspectExec(id)
		if err != nil {
			es.Error = err
			state <- es
			return
		}

		es.Error = nil
		es.ExitCode = i.ExitCode
		es.Running = i.Running

		state <- es

	case <-done:
		return
	}
}

func tgzReader(s *Sync, r io.Reader, st Stream) {

	gz, err := gzip.NewReader(r)
	if err != nil {
		st <- fmt.Sprintf("error: %s", err)
		return
	}

	tr := tar.NewReader(gz)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			st <- fmt.Sprintf("error: %s", err)
			return
		}

		switch header.Typeflag {
		case tar.TypeReg:
			rel, err := filepath.Rel(s.Remote, filepath.Join("/", header.Name))
			if err != nil {
				st <- fmt.Sprintf("error: %s", err)
				return
			}

			local := filepath.Join(s.Local, rel)

			s.lock.Lock()
			s.outgoingBlocks[rel]++
			s.lock.Unlock()

			err = os.MkdirAll(filepath.Dir(local), 0755)
			if err != nil {
				st <- fmt.Sprintf("error: %s", err)
				return
			}

			lf, err := os.Create(local)
			if err != nil {
				st <- fmt.Sprintf("error: %s", err)
				return
			}

			_, err = io.Copy(lf, tr)
			if err != nil {
				st <- fmt.Sprintf("error: %s", err)
				return
			}

			err = lf.Sync()
			if err != nil {
				st <- fmt.Sprintf("error: %s", err)
				return
			}

			err = lf.Close()
			if err != nil {
				st <- fmt.Sprintf("error: %s", err)
				return
			}

			err = os.Chmod(local, os.FileMode(header.Mode))
			if err != nil {
				st <- fmt.Sprintf("error: %s", err)
				return
			}
		}
	}
}
