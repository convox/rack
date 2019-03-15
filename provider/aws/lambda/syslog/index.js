
const MAX_FAILS = 4;

var child_process = require('child_process'),
	go_proc = null,
	done = console.log.bind(console),
	fails = 0;

(function new_go_proc() {

	// pipe stdin/out, blind passthru stderr
	go_proc = child_process.spawn('./main', { stdio: ['pipe', 'pipe', process.stderr] });

	go_proc.on('error', function(err) {
		process.stderr.write("go_proc errored: "+JSON.stringify(err)+"\n");
		if (++fails > MAX_FAILS) {
			process.exit(1); // force container restart after too many fails
		}
		new_go_proc();
		done(err);
	});

	go_proc.on('exit', function(code) {
		process.stderr.write("go_proc exited prematurely with code: "+code+"\n");
		if (++fails > MAX_FAILS) {
			process.exit(1); // force container restart after too many fails
		}
		new_go_proc();
		done(new Error("Exited with code "+code));
	});

	go_proc.stdin.on('error', function(err) {
		process.stderr.write("go_proc stdin write error: "+JSON.stringify(err)+"\n");
		if (++fails > MAX_FAILS) {
			process.exit(1); // force container restart after too many fails
		}
		new_go_proc();
		done(err);
	});

	var data = null;
	go_proc.stdout.on('data', function(chunk) {
		fails = 0; // reset fails
		if (data === null) {
			data = chunk;
		} else {
			data = Buffer.concat([data, chunk]);
		}
		// check for newline ascii char 10
		if (data.length && data[data.length-1] == 10) {
			try {
				var output = JSON.parse(data.toString('UTF-8'));
				data = null;
				done(null, output);
			} catch(err) {
				done(JSON.stringify({
					"error": err.toString('UTF-8'),
					"payload": data.toString('UTF-8')
				}));
			}
		};
	});
})();

exports.handler = function(event, context) {

	// always output to current context's done
	done = context.done.bind(context);

	go_proc.stdin.write(JSON.stringify({
		"event": event,
		"context": context
	})+"\n");

}

