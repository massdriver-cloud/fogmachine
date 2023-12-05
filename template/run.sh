#! /bin/bash
region=us-west-2
stackname=md-test-cf-1234
profile=sandbox

nohup aws --profile $profile cloudformation deploy --stack-name $stackname --template-file ./s3.yaml --parameter-overrides file://s3-values.json --region $region &
watch -n1 aws --profile $profile cloudformation describe-stack-events --stack-name $stackname --output text --query 'StackEvents[*].[ResourceStatus,LogicalResourceId,ResourceType,Timestamp]' --region $region