Does the following:

1. Polls DC for 'Unreachable' nodes and terminates those nodes on EC2.
2. Polls EC2 for states 'terminated' or 'shutting-down' and which have Docker-Cloud-UUID tags and terminates those nodes on Docker Cloud.

Examples:
**A DC node goes into 'Unreachable' state*** Immediately any corresponding ec2 instances will be terminated.
**A BYOH ec2 instance is terminated**: Immediately the corresponding Docker Cloud node will be terminated.

### Usage

Use make to build a docker image tagged as `<docker_username>/docker-cloud-terminator:<git_sha_or_tag>`:

```
make
```

Running locally:

```
export DOCKERCLOUD_AUTH="Basic $(echo -n "<dockercloud_user>:<dockercloud_api_key>" | base64)'"
export AWS_REGION=us-east-1
export AWS_ACCESS_KEY_ID=******
export AWS_SECRET_ACCESS_KEY=******
make run
```