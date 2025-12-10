.PHONY: .git/hooks .git/hooks/ .git/hooks/pre-commit
.git/hooks .git/hooks/ .git/hooks/pre-commit:
	@cp .githooks/* .git/hooks
