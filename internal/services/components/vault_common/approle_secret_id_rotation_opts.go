package vault_common

func WithStaticCidrResolver(cidrs []string) ApproleSecretIdRotationOpts {
	return func(r *ApproleSecretIdRotatorClient) error {
		resolver, err := NewStaticCidrResolver(cidrs)
		if err != nil {
			return err
		}

		r.cidrListResolver = resolver
		return nil
	}
}

func WithStaticCidrTokenResolver(cidrs []string) ApproleSecretIdRotationOpts {
	return func(r *ApproleSecretIdRotatorClient) error {
		resolver, err := NewStaticCidrResolver(cidrs)
		if err != nil {
			return err
		}

		r.tokenCidrResolver = resolver
		return nil
	}
}

func WithDynamicCidrResolver(vaultAddr string) ApproleSecretIdRotationOpts {
	return func(r *ApproleSecretIdRotatorClient) error {
		resolver, err := NewDynamicCidrResolver(vaultAddr)
		if err != nil {
			return err
		}

		r.cidrListResolver = resolver
		return nil
	}
}

func WithDynamicCidrTokenResolver(vaultAddr string) ApproleSecretIdRotationOpts {
	return func(r *ApproleSecretIdRotatorClient) error {
		resolver, err := NewDynamicCidrResolver(vaultAddr)
		if err != nil {
			return err
		}

		r.tokenCidrResolver = resolver
		return nil
	}
}
