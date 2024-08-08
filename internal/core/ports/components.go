package ports

type Services struct {
	K0s               K0s
	Libvirt           Libvirt
	Machine           SystemPowerStatus
	Systemd           Systemd
	Wol               WakeOnLan
	ConditionalReboot ConditionalReboot
	SshSigner         SshPki
	Pki               X509Pki
	SecretSyncer      SecretsReplication
	Packages          SystemPackages
	ReleaseWatcher    ReleaseWatcher
}
