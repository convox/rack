exports.external = function(event, context) {

  process.on('uncaughtException', function(err) {
    return context.done(err);
  });

  console.log('spawning boostrap');
  console.log(JSON.stringify(event));

  var child = require('child_process').spawn('./bootstrap', [JSON.stringify(event)], { stdio:'inherit' })

  // child.stdout.on('data', function(data) {
  //   console.log('data', data);
  // });

  // var child = require('child_process').spawn('ls', ['-la'], {stdio:'pipe'})

  // child.stdin.write(JSON.stringify(event));
  // child.stdin.end();

  // console.log('spawning ls');
  // var child = require('child_process').spawn('ls', ['-la']);

  // child.stdout.pipe(process.stdout);
  // child.stderr.pipe(process.stdout);

  child.on('exit', function(code) {
    if (code !== 0 ) {
      return context.done(new Error("Process exited with non-zero status code: " + code));
    } else {
      context.done(null);
    }
  });
}
