package command

import (
	"fmt"

	"github.com/frantjc/sindri/internal/stoker/stokercr"
	"github.com/frantjc/sindri/internal/stoker/stokercr/api/v1alpha1"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func NewKubectlApproveSteamapps() *cobra.Command {
	var (
		cfgFlags = genericclioptions.NewConfigFlags(true)
		cmd      = &cobra.Command{
			Use:  "kubectl-approve_steamapps",
			Args: cobra.MinimumNArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				var (
					ctx    = cmd.Context()
					cliCfg = cfgFlags.ToRawKubeConfigLoader()
				)

				namespace, ok, err := cliCfg.Namespace()
				if err != nil {
					return err
				} else if !ok || namespace == "" {
					namespace = stokercr.DefaultNamespace
				}

				var (
					scheme = runtime.NewScheme()
				)

				if err := v1alpha1.AddToScheme(scheme); err != nil {
					return err
				}

				restCfg, err := cliCfg.ClientConfig()
				if err != nil {
					return err
				}

				cli, err := client.New(restCfg, client.Options{Scheme: scheme})
				if err != nil {
					return err
				}

				for _, steamappName := range args {
					var (
						steamapp = &v1alpha1.Steamapp{
							ObjectMeta: metav1.ObjectMeta{
								Namespace: namespace,
								Name:      steamappName,
								Annotations: map[string]string{
									stokercr.AnnotationApproved: fmt.Sprint(true),
								},
							},
						}
					)

					if _, err := controllerutil.CreateOrPatch(ctx, cli, steamapp, func() error {
						if steamapp.Annotations == nil {
							steamapp.Annotations = map[string]string{}
						}
						steamapp.Annotations[stokercr.AnnotationApproved] = fmt.Sprint(true)
						return nil
					}); err != nil {
						return err
					}
				}

				return nil
			},
		}
	)

	cfgFlags.AddFlags(cmd.Flags())

	return cmd
}
