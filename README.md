Does the following:

1) Polls DC for 'Unreachable' or 'Terminated' nodes and terminates those nodes on EC2.
2) Polls EC2 for 'Terminated' instances with Docker-Cloud-UUID tags and terminates those nodes on DC.

Examples:
**A DC node goes into 'Unreachable' state*** Immediately any corresponding ec2 instances will be terminated.
**A DC node goes into 'Terminated' state*** Immediately any corresponding ec2 instances will be terminated.
**A BYOH ec2 instance is terminated**: Immediately the corresponding DC node will be terminated.

This means that if a node goes 'Unreachable' in DC it will be terminated by the "round trip". Ie: The ec2 instance will be terminated first, which will in turn trigger a DC node termination. Which will in turn trigger the ec2 instance to be terminated (a no-op since it's already terminated).