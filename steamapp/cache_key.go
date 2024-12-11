package steamapp

import (
	"context"
	"fmt"

	"github.com/frantjc/go-steamcmd"
)

func CacheKey(ctx context.Context, appID int, opts ...Opt) (string, error) {
	o := &Opts{
		platformType: steamcmd.DefaultPlatformType,
	}

	for _, opt := range opts {
		opt(o)
	}

	branchName := DefaultBranchName
	if o.beta != "" {
		branchName = o.beta
	}

	prompt, err := steamcmd.Start(ctx)
	if err != nil {
		return "", err
	}
	defer prompt.Close(ctx)

	if err := prompt.Login(ctx, steamcmd.WithAccount(o.username, o.password)); err != nil {
		return "", err
	}

	appInfo, err := prompt.AppInfoPrint(ctx, appID)
	if err != nil {
		return "", err
	}

	branch, ok := appInfo.Depots.Branches[branchName]
	if !ok {
		return "", fmt.Errorf("branch %s not found", branchName)
	}

	return fmt.Sprint(branch.BuildID), nil
}
