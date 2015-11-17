package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
)

var client = lambda.New(session.New())

func main() {

	build := exec.Command(os.ExpandEnv("$GOROOT/bin/go"), "build", "-o", "main")
	build.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64")
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr

	log.Println("building binary")
	if err := build.Run(); err != nil {
		log.Fatal(err)
	}
	defer os.Remove("main")

	log.Println("building zip")
	var buf bytes.Buffer
	zf := zip.NewWriter(&buf)

	f, _ := zf.Create("index.js")
	f.Write([]byte(indexJS))

	bin, err := os.Open("main")
	if err != nil {
		log.Fatal(err)
	}
	info, err := bin.Stat()
	if err != nil {
		log.Fatal(err)
	}
	header, _ := zip.FileInfoHeader(info)
	f, _ = zf.CreateHeader(header)
	io.Copy(f, bin)
	if err := zf.Close(); err != nil {
		log.Fatal(err)
	}

	if !function_exists(os.Args[1]) {
		log.Fatal("function must be created first")
	}

	log.Println("uploading code")
	resp, err := client.UpdateFunctionCode(&lambda.UpdateFunctionCodeInput{
		FunctionName: aws.String(os.Args[1]),
		Publish:      aws.Bool(true),
		ZipFile:      buf.Bytes(),
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(resp)

}

func function_exists(name string) bool {

	resp, err := client.ListVersionsByFunction(&lambda.ListVersionsByFunctionInput{
		FunctionName: &name,
	})
	return err == nil && len(resp.Versions) > 0

}

// func create() {
// 	type CreateFunctionInput struct {
// 		// The code for the Lambda function.
// 		Code *FunctionCode `type:"structure" required:"true"`

// 		// A short, user-defined function description. Lambda does not use this value.
// 		// Assign a meaningful description as you see fit.
// 		Description *string `type:"string"`

// 		// The name you want to assign to the function you are uploading. You can specify
// 		// an unqualified function name (for example, "Thumbnail") or you can specify
// 		// Amazon Resource Name (ARN) of the function (for example, "arn:aws:lambda:us-west-2:account-id:function:ThumbNail").
// 		// AWS Lambda also allows you to specify only the account ID qualifier (for
// 		// example, "account-id:Thumbnail"). Note that the length constraint applies
// 		// only to the ARN. If you specify only the function name, it is limited to
// 		// 64 character in length. The function names appear in the console and are
// 		// returned in the ListFunctions API. Function names are used to specify functions
// 		// to other AWS Lambda APIs, such as Invoke.
// 		FunctionName *string `min:"1" type:"string" required:"true"`

// 		// The function within your code that Lambda calls to begin execution. For Node.js,
// 		// it is the module-name.export value in your function. For Java, it can be
// 		// package.class-name::handler or package.class-name. For more information,
// 		// see Lambda Function Handler (Java) (http://docs.aws.amazon.com/lambda/latest/dg/java-programming-model-handler-types.html).
// 		Handler *string `type:"string" required:"true"`

// 		// The amount of memory, in MB, your Lambda function is given. Lambda uses this
// 		// memory size to infer the amount of CPU and memory allocated to your function.
// 		// Your function use-case determines your CPU and memory requirements. For example,
// 		// a database operation might need less memory compared to an image processing
// 		// function. The default value is 128 MB. The value must be a multiple of 64
// 		// MB.
// 		MemorySize *int64 `min:"128" type:"integer"`

// 		// This boolean parameter can be used to request AWS Lambda to create the Lambda
// 		// function and publish a version as an atomic operation.
// 		Publish *bool `type:"boolean"`

// 		// The Amazon Resource Name (ARN) of the IAM role that Lambda assumes when it
// 		// executes your function to access any other Amazon Web Services (AWS) resources.
// 		// For more information, see AWS Lambda: How it Works (http://docs.aws.amazon.com/lambda/latest/dg/lambda-introduction.html)
// 		Role *string `type:"string" required:"true"`

// 		// The runtime environment for the Lambda function you are uploading. Currently,
// 		// Lambda supports "java" and "nodejs" as the runtime.
// 		Runtime *string `type:"string" required:"true" enum:"Runtime"`

// 		// The function execution time at which Lambda should terminate the function.
// 		// Because the execution time has cost implications, we recommend you set this
// 		// value based on your expected execution time. The default is 3 seconds.
// 		Timeout *int64 `min:"1" type:"integer"`
// 	}

// }

const (
	indexJS = `
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
			var output = JSON.parse(data.toString('UTF-8'));
			data = null;
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

};
`
)
