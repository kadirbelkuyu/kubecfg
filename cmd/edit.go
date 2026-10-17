package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kadirbelkuyu/kubecfg/internal/application"
	"github.com/kadirbelkuyu/kubecfg/internal/domain"
	"github.com/kadirbelkuyu/kubecfg/internal/infrastructure"
	"github.com/kadirbelkuyu/kubecfg/internal/ui"
)

var (
	editFrom  string
	editPrint bool
)

var editCmd = &cobra.Command{
	Use:     "edit [context-name]",
	Aliases: []string{"update"},
	Short:   "Edit a context definition",
	Long: "Update the definition of an existing context in place.\n\n" +
		"The context entry together with the cluster and user it references is opened in an editor as a single-context kubeconfig. " +
		"Saving applies the document, so renewed certificates, a moved API server, or changed credentials can be updated without removing and re-adding the context.\n\n" +
		"Use --from to apply a definition from a kubeconfig file instead of opening an editor. Without a context name the current context is edited.\n\n" +
		"The editor is taken from KUBECFG_EDITOR, KUBE_EDITOR, VISUAL, or EDITOR.",
	Example: `  kubecfg edit
  kubecfg edit production
  kubecfg edit production --from ./renewed-prod.yaml
  kubecfg edit production --print`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		contextName := ""
		if len(args) == 1 {
			contextName = args[0]
		}

		if editPrint {
			printContextDefinition(contextName)
			return
		}

		update, err := updateContextDefinition(contextName)
		if err != nil {
			if errors.Is(err, domain.ErrEditAborted) {
				printSuccess("Canceled")
				return
			}
			printError(err)
			return
		}

		if update.IsEmpty() {
			printInfo(fmt.Sprintf("Context '%s' is unchanged", ui.ContextName(update.ContextName)))
			return
		}

		printSuccess(fmt.Sprintf("Context '%s' updated (%s)",
			ui.ContextName(update.ContextName),
			strings.Join(update.Changed, ", ")))
	},
}

func printContextDefinition(contextName string) {
	definition, err := service.GetContextDefinition(kubeconfigPath, contextName)
	if err != nil {
		printError(err)
		return
	}

	fmt.Print(definition)
}

func updateContextDefinition(contextName string) (*application.ContextUpdate, error) {
	if editFrom != "" {
		return service.UpdateContextFromFile(kubeconfigPath, contextName, editFrom)
	}

	return service.EditContextDefinition(kubeconfigPath, contextName, infrastructure.NewExternalEditor())
}

func init() {
	editCmd.Flags().StringVar(&editFrom, "from", "", "apply the definition from a kubeconfig file instead of opening an editor")
	editCmd.Flags().BoolVar(&editPrint, "print", false, "print the context definition without editing it")
	editCmd.MarkFlagsMutuallyExclusive("from", "print")
	rootCmd.AddCommand(editCmd)
}
