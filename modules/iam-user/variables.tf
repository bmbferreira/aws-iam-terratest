variable "user_name" {
    type = string
    default = "test"
}

variable "allowed_ips" {
    type = list(string)
    default = []
}

variable "iam_policy_name" {
    type = string
    default = "VpcCidrRestricted"
}
