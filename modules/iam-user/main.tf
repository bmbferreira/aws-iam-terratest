resource "aws_iam_user" "user" {
  name = var.user_name
}

resource "aws_iam_access_key" "iam_user_key" {
  user = aws_iam_user.user.name
}

resource "aws_iam_user_policy_attachment" "restricted" {
  user       = aws_iam_user.user.name
  policy_arn = aws_iam_policy.vpc.arn
}

resource "aws_iam_policy" "vpc" {
  name        = var.iam_policy_name
  path        = "/"
  description = "Limits environment keys to VPC origin"

  policy = data.aws_iam_policy_document.vpc.json
}


data "aws_iam_policy_document" "vpc" {
  statement {
    sid           = ""
    effect        = "Deny"
    not_resources = ["arn:aws:s3:::test"]

    #not_actions = [
    actions = [
      "kms:*",
      "sns:*",
      "sqs:*",
      "cloudfront:CreateInvalidation",
    ]

    condition {
      test     = "NotIpAddress"
      variable = "aws:SourceIp"

      values = var.allowed_ips
    }
  }
}

resource "aws_iam_user_policy" "allow_sqs" {
  name = "allow_sqs"
  user = aws_iam_user.user.name

  policy = data.aws_iam_policy_document.allow_sqs.json
}

data "aws_iam_policy_document" "allow_sqs" {
  statement {
    sid           = ""
    effect        = "Allow"
    resources = ["*"]

    actions = [
      "sqs:*",
    ]
  }
}
