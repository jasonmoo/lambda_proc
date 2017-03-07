#Lambda_proc

This project is experimental. Event though it passes [tests](https://github.com/jasonmoo/lambda_proc/blob/master/lambda_proc_test.go) it should still be considered experimental.

This project allows you to leverage [container reuse](https://aws.amazon.com/blogs/compute/container-reuse-in-lambda/) in AWS lambda by starting '[go proc](https://github.com/jasonmoo/lambda_proc/blob/master/example/index.js#L12)' as a companion process, communicating via stdin/stdout with the parent nodejs process. '[Run](https://github.com/jasonmoo/lambda_proc/blob/master/lambda_proc.go#L64)' takes a handler function and runs it once per json [payload](https://github.com/jasonmoo/lambda_proc/blob/master/lambda_proc.go#L25) received from the parent nodejs process. 

Lambda containers may be frozen and thawed between invocations. This allows a 'go proc' to live for many requests, allowing [memoization](https://en.wikipedia.org/wiki/Memoization), startup cost elimination, and performance benefits. 

Observed lambda run times are between .5ms-15ms after first invocation.

"[index.js](https://github.com/jasonmoo/lambda_proc/blob/master/example/index.js)" has much more robust error handling than typical examples and will restart the go companion process ('go_proc') if it prematurely dies, a syscall returns a failure, or other error scenarios.

The example folder '[build.sh](https://github.com/jasonmoo/lambda_proc/blob/master/example/build.sh)' script and code are all **complete** examples of a working lambda.


License: MIT 2015