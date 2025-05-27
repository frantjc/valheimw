package command

import (
	"fmt"
	"strings"

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
		all      bool
		cfgFlags = genericclioptions.NewConfigFlags(true)
		cmd      = &cobra.Command{
			Use: "kubectl-approve_steamapps",
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

				var (
					steamapps = &v1alpha1.SteamappList{}
				)

				switch {
				case all:
					if len(args) > 0 {
						return fmt.Errorf("%s and --all are mutually exclusive", strings.Join(args, " "))
					}

					if err := cli.List(ctx, steamapps, &client.ListOptions{
						Namespace: namespace,
					}); err != nil {
						return err
					}
				case len(args) == 0:
					return fmt.Errorf("names or --all are required")
				default:
					for _, steamappName := range args {
						steamapps.Items = append(steamapps.Items, v1alpha1.Steamapp{
							ObjectMeta: metav1.ObjectMeta{
								Namespace:   namespace,
								Name:        steamappName,
								Annotations: map[string]string{},
							},
						})
					}
				}

				for _, steamapp := range steamapps.Items {
					if steamapp.Annotations == nil {
						steamapp.Annotations = map[string]string{}
					}
					steamapp.Annotations[stokercr.AnnotationApproved] = fmt.Sprint(true)

					if _, err := controllerutil.CreateOrPatch(ctx, cli, &steamapp, func() error {
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
	cmd.Flags().BoolVarP(&all, "all", "A", false, "Approve all Steamapps")

	return cmd
}
