package download

import (
	"io/fs"
	"log"
	"spear/config"
	"spear/utils/client_vfs"
)

func SpearDownload(spearConfig *config.SpearConfig, downloadParams *config.DownloadParams) {

	log.Printf("Download called with params: %+v", downloadParams)
	identity, ok := spearConfig.Identities["default"]
	if !ok {
		log.Fatal("default identity not found")
	}
	client := client_vfs.NewSpearClient(spearConfig, downloadParams, &identity)
	spearFS := client_vfs.NewSpearFS(client, "/")

	fs.WalkDir(spearFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Printf("error walking to %s, %v: %v", path, d.IsDir(), err)
			return err
		}
		log.Printf("Visited: %s", path)
		return nil
	})
	/*err = os.CopyFS(downloadParams.OutDir, spearFS)
	if err != nil {
		log.Fatalf("failed to copy from spearFS to outdir: %v", err)
	}*/

}
