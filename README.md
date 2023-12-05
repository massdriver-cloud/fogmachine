# Fogmachine

A CLI tool for running cloudformation in CI. Cloudformation verbose logging contains data that is noise when trying to determine what resources are created and what has happened during and after execution. This CLI aims to provide detailed output when running cloudformation configuration.

## Setup
- Open vscode with the dev container.
- Add AWS_SECRET_ACCESS_KEY and AWS_ACCESS_KEY_ID to env
- ./fogmachine apply --package-name md-test-cf-1234 --region us-west-2 --template-path template/s3.yaml --parameter-path template/s3-values.json 
- Change value in s3-values.json from 1234 -> 12345 and back to get executions

## TODO
* Set polling interval
* Set Timeout
* Add destroy (subset of what already exists)

