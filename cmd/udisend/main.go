package udisend

import (
	"context"
	"log"
	"udisend/config"
	"udisend/internal/network"
	"udisend/pkg/closer"
	"udisend/pkg/crypt"
)

func main() {
	cfg := config.NewConfig()
	privateAuth, pubAuth, err := crypt.LoadOrGenerateKeys(cfg.PrivateAuthKeyFile, cfg.PrivateAuthKeyFile)
	if err != nil {
		log.Fatalf("error load auth keys: %v", err)
	}
	privateRSA, publicRSA, err := crypt.GenerateRSAKeys()
	if err != nil {
		log.Fatalf("error generate rsa keys: %v", err)
	}
	nw := network.New(cfg.ID, pubAuth, privateAuth, func(b []byte) ([]byte, error) {
		return crypt.DecryptMessage(b, privateRSA)
	}, publicRSA)

	ctx := context.Background()

	closer.Add(func() error {
		ctx.Done()
		return nil
	})

	nw.Run(ctx)
}
