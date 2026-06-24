package installer

func Package(pkgs ...string) error {
	if len(pkgs) == 0 {
		return nil
	}

	args := make([]string, 0, len(pkgs)+1)
	args = append(args, "install", "-y")
	args = append(args, pkgs...)

	return run("dnf", args...)
}
