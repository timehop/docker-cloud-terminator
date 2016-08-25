# Docker Cloud Terminator

## What

This repo provides a Docker Cloud stack that helps manage the lifecycle of ["Bring Your Own Host"](https://docs.docker.com/docker-cloud/infrastructure/byoh/) nodes. Specifically, it keeps the state of an EC2 instance in sync with its representation as a node on Docker Cloud. 

In other words, when a node become "Unreachable", this service terminates both it and the EC2 instance it represents. Likewise, if an EC2 instances shuts down, this service terminates the node on Docker Cloud.

[![Deploy to Docker Cloud](https://files.cloud.docker.com/images/deploy-to-dockercloud.svg)](https://cloud.docker.com/stack/deploy/)

The "Deploy to Cloud" button above may be used to quickly get up and running. Refer to the comments in `docker-cloud.yml` for configuration details.

## Why

When using BYOH nodes, Docker Cloud has no system for inspecting and terminating nodes that become "Unreachable". This is a solution for that problem.

### AWS Credentials

The AWS credentials you configure the service with will need to have s3 write access to the bucket you configure, as well as access for creating CloudFormation stacks. Here is a sample policy you may use. Replace `${S3_BUCKET}` with the configured bucket name:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:DescribeInstances",
        "ec2:DescribeTags",
        "ec2:TerminateInstances"
      ],
      "Resource": "*"
    }
  ]
}
```


