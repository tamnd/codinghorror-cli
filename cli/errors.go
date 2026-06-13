package cli

func mapFetchErr(err error) error {
	if err == nil {
		return nil
	}
	return codeError(exitError, err)
}
