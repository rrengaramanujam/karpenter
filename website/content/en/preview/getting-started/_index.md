---
title: "Getting Started"
linkTitle: "Getting Started"
weight: 1
description: >
  Choose from different methods to get started with Karpenter
cascade:
  type: docs
---


To get started with Karpenter, the [Getting Started with Karpenter]({{< relref "getting-started-with-karpenter" >}}) guide provides an end-to-end procedure for creating a cluster (with `eksctl`) and adding Karpenter.
If you prefer, the following instructions use Terraform to create a cluster and add Karpenter:

* [Amazon EKS Blueprints for Terraform](https://aws-ia.github.io/terraform-aws-eks-blueprints): Follow a basic [Getting Started](https://aws-ia.github.io/terraform-aws-eks-blueprints/v4.18.0/getting-started/) guide and also add modules and add-ons. This includes a [Karpenter](https://aws-ia.github.io/terraform-aws-eks-blueprints/v4.18.0/add-ons/karpenter/) add-on that lets you bypass the instructions in this guide for setting up Karpenter.

Although not supported, you could also try Karpenter on other Kubernetes distributions running on AWS. For example:

* [kOps](https://kops.sigs.k8s.io/operations/karpenter/): These instructions describe how to create a kOps Kubernetes cluster in AWS that includes Karpenter.

Learn more about Karpenter and how to get started below.

* [Karpenter EKS Best Practices](https://aws.github.io/aws-eks-best-practices/karpenter/) guide
* [EC2 Spot Workshop for Karpenter](https://ec2spotworkshops.com/karpenter.html)
* [EKS Karpenter Workshop](https://www.eksworkshop.com/docs/autoscaling/compute/karpenter/)
* [Advanced EKS Immersion Karpenter Workshop](https://catalog.workshops.aws/eks-advanced/karpenter/)
