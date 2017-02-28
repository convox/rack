exports.external = function(event, context) {
  console.log('event', JSON.stringify(event));
  console.log('context', JSON.stringify(context));

  process.on('uncaughtException', function(err) {
    return context.done(err);
  });

  var child = require('child_process').spawn('./main', [JSON.stringify(event)]);

  child.stdout.on('data', function(data) {
    console.log(data.toString());
  });

  child.stderr.on('data', function(data) {
    console.log(data.toString());
  });

  child.on('close', function(code) {
    if (code !== 0) {
      return context.done(new Error("Process exited with non-zero status code: " + code));
    }
    context.done(null);
  });
}
