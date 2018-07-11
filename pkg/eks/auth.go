package eks

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/pkg/errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"

	"github.com/heptio/authenticator/pkg/token"
	"github.com/kubicorn/kubicorn/pkg/logger"

	clientset "k8s.io/client-go/kubernetes"

	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"k8s.io/kops/pkg/pki"
	"k8s.io/kops/upup/pkg/fi/utils"
)

func (c *ClusterProvider) getKeyPairName(fingerprint *string) string {
	keyNameParts := []string{"eksctl", c.cfg.ClusterName}
	if fingerprint != nil {
		keyNameParts = append(keyNameParts, *fingerprint)
	}
	return strings.Join(keyNameParts, "-")
}

func (c *ClusterProvider) getKeyPair(name string) (*ec2.KeyPairInfo, error) {
	input := &ec2.DescribeKeyPairsInput{
		KeyNames: aws.StringSlice([]string{name}),
	}
	output, err := c.svc.ec2.DescribeKeyPairs(input)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot find EC2 key pair %q", name)
	}
	if len(output.KeyPairs) != 1 {
		logger.Debug("output = %#v", output)
		return nil, fmt.Errorf("unexpected number of key pairs found (expected: 1, got: %d)", len(output.KeyPairs))
	}
	return output.KeyPairs[0], nil
}

func (c *ClusterProvider) tryExistingSSHPublicKeyFromPath() error {
	logger.Info("SSH public key file %q does not exist; will assume existing EC2 key pair", c.cfg.SSHPublicKeyPath)
	existing, err := c.getKeyPair(c.cfg.SSHPublicKeyPath)
	if err != nil {
		return err
	}
	c.cfg.keyName = *existing.KeyName
	logger.Info("found EC2 key pair %q", c.cfg.keyName)
	return nil
}

func (c *ClusterProvider) importSSHPublicKeyIfNeeded() error {
	fingerprint, err := pki.ComputeAWSKeyFingerprint(string(c.cfg.SSHPublicKey))
	if err != nil {
		return err
	}
	c.cfg.keyName = c.getKeyPairName(&fingerprint)
	existing, err := c.getKeyPair(c.cfg.keyName)
	if err != nil {
		if strings.HasPrefix(err.Error(), "cannot find EC2 key pair") {
			input := &ec2.ImportKeyPairInput{
				KeyName:           &c.cfg.keyName,
				PublicKeyMaterial: c.cfg.SSHPublicKey,
			}
			logger.Info("importing SSH public key %q as %q", c.cfg.SSHPublicKeyPath, c.cfg.keyName)
			if _, err := c.svc.ec2.ImportKeyPair(input); err != nil {
				return errors.Wrap(err, "importing SSH public key")
			}
			return nil
		}
		return errors.Wrap(err, "checking existing key pair")
	}
	if *existing.KeyFingerprint != fingerprint {
		return fmt.Errorf("SSH public key %s already exists, but fingerprints don't match (exected: %q, got: %q)", c.cfg.keyName, fingerprint, *existing.KeyFingerprint)
	}
	logger.Debug("SSH public key %s already exists", c.cfg.keyName)
	return nil
}

func (c *ClusterProvider) LoadSSHPublicKey() error {
	c.cfg.SSHPublicKeyPath = utils.ExpandPath(c.cfg.SSHPublicKeyPath)
	sshPublicKey, err := ioutil.ReadFile(c.cfg.SSHPublicKeyPath)
	if err != nil {
		if os.IsNotExist(err) {
			// if file not found – try to use existing EC2 key pair
			return c.tryExistingSSHPublicKeyFromPath()
		}
		return errors.Wrap(err, fmt.Sprintf("reading SSH public key file %q", c.cfg.SSHPublicKeyPath))
	}
	// on successfull read – import it
	c.cfg.SSHPublicKey = sshPublicKey
	return c.importSSHPublicKeyIfNeeded()
}

func (c *ClusterProvider) MaybeDeletePublicSSHKey() {
	existing, err := c.svc.ec2.DescribeKeyPairs(&ec2.DescribeKeyPairsInput{})
	if err != nil {
		logger.Debug("cannot describe keys: %v", err)
		return
	}
	matching := []*string{}
	prefix := c.getKeyPairName(nil)
	logger.Debug("existing = %#v", existing)
	for _, e := range existing.KeyPairs {
		if strings.HasPrefix(*e.KeyName, prefix) {
			nameParts := strings.Split(*e.KeyName, "-")
			logger.Debug("existing key %q matches prefix", *e.KeyName)
			if nameParts[len(nameParts)-1] == *e.KeyFingerprint {
				logger.Debug("existing key %q matches fingerprint", *e.KeyName)
				matching = append(matching, e.KeyName)
			}
		}
	}
	if len(matching) > 1 {
		logger.Debug("too many matching keys, will not delete any")
		return
	}
	if len(matching) == 1 {
		input := &ec2.DeleteKeyPairInput{
			KeyName: matching[0],
		}
		logger.Debug("deleting key %q", *matching[0])
		c.svc.ec2.DeleteKeyPair(input)
	}
}

func (c *ClusterProvider) getUsername() string {
	usernameParts := strings.Split(c.svc.arn, "/")
	if len(usernameParts) > 1 {
		return usernameParts[len(usernameParts)-1]
	}
	return "iam-root-account"
}

type ClientConfig struct {
	Client      *clientcmdapi.Config
	Cluster     *ClusterConfig
	ClusterName string
	ContextName string
	roleARN     string
	sts         stsiface.STSAPI
}

// based on "k8s.io/kubernetes/cmd/kubeadm/app/util/kubeconfig"
// these are small, so we can copy these, and no need to deal with k/k as dependency
func (c *ClusterProvider) NewClientConfig() (*ClientConfig, error) {
	clusterName := fmt.Sprintf("%s.%s.eksctl.io", c.cfg.ClusterName, c.cfg.Region)
	contextName := fmt.Sprintf("%s@%s", c.getUsername(), clusterName)

	clientConfig := &ClientConfig{
		Cluster: c.cfg,
		Client: &clientcmdapi.Config{
			Clusters: map[string]*clientcmdapi.Cluster{
				clusterName: {
					Server: c.cfg.MasterEndpoint,
					CertificateAuthorityData: c.cfg.CertificateAuthorityData,
				},
			},
			Contexts: map[string]*clientcmdapi.Context{
				contextName: {
					Cluster:  clusterName,
					AuthInfo: contextName,
				},
			},
			AuthInfos: map[string]*clientcmdapi.AuthInfo{
				contextName: &clientcmdapi.AuthInfo{},
			},
			CurrentContext: contextName,
		},
		ClusterName: clusterName,
		ContextName: contextName,
		roleARN:     c.svc.arn,
		sts:         c.svc.sts,
	}

	return clientConfig, nil
}

func (c *ClientConfig) WithExecHeptioAuthenticator() *ClientConfig {
	clientConfigCopy := *c

	x := clientConfigCopy.Client.AuthInfos[c.ContextName]
	x.Exec = &clientcmdapi.ExecConfig{
		APIVersion: "client.authentication.k8s.io/v1alpha1",
		Command:    "heptio-authenticator-aws",
		Args:       []string{"token", "-i", c.Cluster.ClusterName},
		/*
			Args:       []string{"token", "-i", c.Cluster.ClusterName, "-r", c.roleARN},
		*/
	}

	if len(c.Cluster.Profile) > 0 {
		profileVar := &clientcmdapi.ExecEnvVar{Name: "AWS_PROFILE", Value: c.Cluster.Profile}
		x.Exec.Env = []clientcmdapi.ExecEnvVar{*profileVar}
	}

	return &clientConfigCopy
}

func (c *ClientConfig) WithEmbeddedToken() (*ClientConfig, error) {
	clientConfigCopy := *c

	gen, err := token.NewGenerator()
	if err != nil {
		return nil, errors.Wrap(err, "could not get token generator")
	}

	tok, err := gen.GetWithSTS(c.Cluster.ClusterName, c.sts.(*sts.STS))
	if err != nil {
		return nil, errors.Wrap(err, "could not get token")
	}

	x := c.Client.AuthInfos[c.ContextName]
	x.Token = tok

	return &clientConfigCopy, nil
}

func (c *ClientConfig) NewClientSet() (*clientset.Clientset, error) {
	clientConfig, err := clientcmd.NewDefaultClientConfig(*c.Client, &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create API client configuration from client config")
	}

	client, err := clientset.NewForConfig(clientConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create API client")
	}
	return client, nil
}

func (c *ClientConfig) NewClientSetWithEmbeddedToken() (*clientset.Clientset, error) {
	clientConfig, err := c.WithEmbeddedToken()
	if err != nil {
		return nil, errors.Wrap(err, "creating Kubernetes client config with embedded token")
	}
	clientSet, err := clientConfig.NewClientSet()
	if err != nil {
		return nil, errors.Wrap(err, "creating Kubernetes client")
	}
	return clientSet, nil
}
