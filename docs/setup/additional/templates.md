# General guidelines for Templates

1. Avoid custom templates specific to the use case. Generalise the templates.
2. Avoid Hardcoding & Use Namespaces Smartly
3. Use Template Helpers (_helpers.tpl) to extract common patterns (e.g., labels, resource names)

## Rolebinding Templates

There are two definitions of rolebinding templates in the chart

1. HBase Operator manager Rolebinding
2. List of additional rolebindings if required

### HBase operator manager rolebinding

> <b>Purpose:</b> To bind the cluster role for hbase-operator-manager to service account in the
> namespace
> specified

It is defined as `com.flipkart.hbaseoperator.rolebindings`.
This rolebinding binds the ClusterRole to a service account. This service account is present in the
namespace specified as operatorNamespace. Note that the service account may or may not present in
the
same namespace.

### List of additional rolebindings

> <b>Purpose:</b> To serve specific use cases where different roles and rolebindings are needed over
> HBase
> clusters

It is defined as `com.flipkart.hbaseresources.rolebindings`.
This template definition is purposefully kept separate from the operator rolebinding template
definition. It serves specific use cases
other than the operator.
These rolebindings bind Role/ ClusterRole having different set of permissions from the operator role
and needed to be bound in different namespaces.
One role can also be bound to multiple service accounts using this template. Specify the service
account and their namespaces in the `Kind` section of the rolebinding template.

## Service account template

It is defined as `com.flipkart.hbaseresources.serviceaccounts`.
Service accounts can be created for specific use cases on a certain namespace. Multiple service
accounts creation is supported using the mentioned template.
> [!IMPORTANT]
> Operator service account is created on the operator namespace and the core cluster is
> present on a different namespace. Hence, it is not required to specify operator service account in
> the values inside the additional service accounts.

## Role Template

It is defined as `com.flipkart.hbaseresources.roles`.
It will create the new roles with the set of permissions mentioned in the values file. This is also
kept separate from the hbase operator role. It also caters specific use cases related to hbase
managed service.

## Guidelines for specifying service accounts, roles and rolebindings in values.yaml file

1. Classify the service accounts that needed to be created on core cluster vs tenant cluster.
2. Classify the rolebindings that needed to be created on core cluster vs the tenant cluster.

> [!WARNING]
> The responsibility of adding correct service accounts and rolebinding lies completely on the user.