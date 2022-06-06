package k8s

import (
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

func ReadServiceAccountToken(saName string, namespace string) (string, error) {
	clientSet, err := NewInClusterClient()
	if err != nil {
		panic(err.Error())
	}
	ctx := context.Background()

	sa, err := clientSet.CoreV1().ServiceAccounts(namespace).Get(ctx, saName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to find service account %s", err)
	}
	fmt.Println(sa.Name)

	var saSecret string
	for _, s := range sa.Secrets {
		if strings.Contains(s.Name, "-token-") {
			saSecret = s.Name
			break
		}
	}
	fmt.Println("secret", saSecret)

	secret, err := clientSet.CoreV1().Secrets(namespace).Get(ctx, saSecret, metav1.GetOptions{})
	if err != nil {
		fmt.Println("err", err)
	}

	token, ok := secret.Data["token"]
	if !ok {
		return "", fmt.Errorf("failed to look up token from sa secret")
	}

	return string(token), nil
}
