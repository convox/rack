package helpers

func AtomStatus(status string) string {
	switch status {
	case "Failed":
		return "failed"
	case "Rollback":
		return "rollback"
	case "Deadline", "Error", "Pending", "Running":
		return "updating"
	default:
		return "running"
	}
}
