module "remote_module" {
  source = "git::https://github.com/JackFlukinger/rover.git//example/random-test/random-name?ref=fix-remote-repo"

  max_length = "3"

}