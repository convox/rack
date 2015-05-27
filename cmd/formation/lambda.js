exports.external = function(event, context) {

  process.on('uncaughtException', function(err) {
    return context.done(err);
  });

  var child = require('child_process').spawn('./formation', []);

  child.stdin.write(JSON.stringify(event));
  child.stdin.end();

  child.stderr.pipe(process.stdout);

  var output = "";

  child.stdout.on('data', function(data) {
    output += data.toString();
  });

  child.on('close', function(code) {
    if (code !== 0 ) {
      return context.done(new Error("Process exited with non-zero status code: " + code));
    } else {
      context.done(null, output);
    }
  });
}
