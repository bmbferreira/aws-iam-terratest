output "aws_iam_access_key_id" {
    value = aws_iam_access_key.iam_user_key.id
    sensitive = true
}

output "aws_iam_access_key_secret" {
    value = aws_iam_access_key.iam_user_key.secret
    sensitive = true
}
