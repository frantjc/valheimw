package api

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
)

func NewExecHandlerWithPortFromEnv(ctx context.Context, name string, args ...string) (http.Handler, *exec.Cmd, error) {
	var (
		cmd = exec.CommandContext(ctx, name, args...)
	)

	lis, err := net.Listen("tcp", "127.0.0.1:")
	if err != nil {
		return nil, nil, err
	}

	_, port, err := net.SplitHostPort(lis.Addr().String())
	if err != nil {
		return nil, nil, err
	}

	target, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%s", port))
	if err != nil {
		return nil, nil, err
	}

	cmd.Env = append(os.Environ(), fmt.Sprintf("PORT=%s", port))

	if err = lis.Close(); err != nil {
		return nil, nil, err
	}

	return httputil.NewSingleHostReverseProxy(target), cmd, nil
}