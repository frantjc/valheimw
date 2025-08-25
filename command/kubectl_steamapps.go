package command

import (
	"fmt"
	"strings"

	"github.com/frantjc/sindri/internal/api"
	"github.com/frantjc/sindri/internal/api/v1alpha1"
	"github.com/frantjc/sindri/internal/controller"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func NewKubectlSteamapps() *cobra.Command {
	var (
		cfgFlags = genericclioptions.NewConfigFlags(true)
		cmd      = &cobra.Command{
			Use:     "steamapps",
			Aliases: []string{"steamapp", "sa"},
		}
	)

	cfgFlags.AddFlags(cmd.PersistentFlags())
	cmd.AddCommand(NewKubectlSteamappsApprove(cfgFlags))
	cmd.AddCommand(NewKubectlSteamappsLogs(cfgFlags))

	return cmd
}

func NewKubectlSteamappsApprove(cfgFlags *genericclioptions.ConfigFlags) *cobra.Command {
	var (
		all bool
		cmd = &cobra.Command{
			Use:     "approve",
			Aliases: []string{"ap"},
			RunE: func(cmd *cobra.Command, args []string) error {
				var (
					ctx    = cmd.Context()
					cliCfg = cfgFlags.ToRawKubeConfigLoader()
				)

				namespace, ok, err := cliCfg.Namespace()
				if err != nil {
					return err
				} else if !ok || namespace == "" {
					namespace = controller.DefaultNamespace
				}

				scheme, err := api.NewScheme()
				if err != nil {
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

				steamapps := &v1alpha1.SteamappList{}

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
					} else if steamapp.Annotations[controller.AnnotationApproved] == fmt.Sprint(true) {
						if _, err = fmt.Fprintf(cmd.OutOrStdout(), "steamapp %q unchanged\n", steamapp.Name); err != nil {
							return err
						}

						continue
					}

					steamapp.Annotations[controller.AnnotationApproved] = fmt.Sprint(true)

					if _, err := controllerutil.CreateOrPatch(ctx, cli, &steamapp, func() error {
						if steamapp.Annotations == nil {
							steamapp.Annotations = map[string]string{}
						}
						steamapp.Annotations[controller.AnnotationApproved] = fmt.Sprint(true)
						return nil
					}); err != nil {
						return err
					}

					if _, err = fmt.Fprintf(cmd.OutOrStdout(), "steamapp %q approved\n", steamapp.Name); err != nil {
						return err
					}
				}

				return nil
			},
		}
	)

	cmd.Flags().BoolVarP(&all, "all", "A", false, "Approve all Steamapps")

	return cmd
}

func NewKubectlSteamappsLogs(cfgFlags *genericclioptions.ConfigFlags) *cobra.Command {
	var (
		cmd = &cobra.Command{
			Use:     "logs",
			Aliases: []string{"log"},
			Args:    cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				var (
					ctx    = cmd.Context()
					cliCfg = cfgFlags.ToRawKubeConfigLoader()
				)

				namespace, ok, err := cliCfg.Namespace()
				if err != nil {
					return err
				} else if !ok || namespace == "" {
					namespace = controller.DefaultNamespace
				}

				scheme, err := api.NewScheme()
				if err != nil {
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

				cm := &corev1.ConfigMap{}

				if err := cli.Get(ctx, client.ObjectKey{Namespace: namespace, Name: args[0]}, cm); err != nil {
					return err
				}

				if _, err = fmt.Fprint(cmd.OutOrStdout(), cm.Data[controller.LogKey]); err != nil {
					return err
				}

				return nil
			},
		}
	)

	return cmd
}
