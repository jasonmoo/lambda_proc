
const MAX_FAILS = 4;

var child_process = require('child_process'),
	go_proc = null,
	done = console.log.bind(console),
	fails = 0;

// Taken from Prototype. https://github.com/sstephenson/prototype
function isJSON(str) {
	if (str == "") return false;

	str = str.replace(/\\(?:["\\\/bfnrt]|u[0-9a-fA-F]{4})/g, '@');
	str = str.replace(/"[^"\\\n\r]*"|true|false|null|-?\d+(?:\.\d*)?(?:[eE][+\-]?\d+)?/g, ']');
	str = str.replace(/(?:^|:|,)(?:\s*\[)+/g, '');

	return (/^[\],:{}\s]*$/).test(str);
}

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
			data = new Buffer(chunk);
		} else {
			data.write(chunk);
		}
		// check for newline ascii char 10
		if (data.length && data[data.length-1] == 10) {
			// Get the data as a string, then reset.
			var str = data.toString('UTF-8');
			data = null;
			// If this isn't json, it is just a log.
			if (!isJSON(str)) {
				console.log(str.substring(0, str.length-2));
				return;
			}
			// This is json response, we are done.
			var output = JSON.parse(str);
			done(null, output);
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

