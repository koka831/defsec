
### IAM policy should avoid use of wildcards and instead apply the principle of least privilege

You should use the principle of least privilege when defining your IAM policies. This means you should specify each exact permission required without using wildcards, as this could cause the granting of access to certain undesired actions, resources and principals.

### Impact
Overly permissive policies may grant access to sensitive resources

<!-- DO NOT CHANGE -->
{{ remediationActions }}

### Links
- https://docs.aws.amazon.com/IAM/latest/UserGuide/best-practices.html
        