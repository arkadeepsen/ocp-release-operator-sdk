// Copyright 2019 The Operator-SDK Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package installer

import (
	"context"
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

const (
	// TODO: switch back to latest once olm fixes their releases
	// https://github.com/operator-framework/operator-lifecycle-manager/issues/3419
	DefaultVersion = "0.28.0"
	DefaultTimeout = time.Minute * 2
	// DefaultOLMNamespace is the namespace where OLM is installed
	DefaultOLMNamespace = "olm"
)

type Manager struct {
	Client       *Client
	Version      string
	Timeout      time.Duration
	OLMNamespace string
	once         sync.Once
}

func (m *Manager) initialize() (err error) {
	m.once.Do(func() {
		if m.Client == nil {
			cfg, cerr := config.GetConfig()
			if cerr != nil {
				err = fmt.Errorf("failed to get Kubernetes config: %v", cerr)
				return
			}

			client, cerr := ClientForConfig(cfg)
			if cerr != nil {
				err = fmt.Errorf("failed to create manager client: %v", cerr)
				return
			}
			m.Client = client
		}
		if m.Timeout <= 0 {
			m.Timeout = DefaultTimeout
		}
		if m.OLMNamespace == "" {
			m.OLMNamespace = DefaultOLMNamespace
		}
	})
	return err
}

func (m *Manager) Install() error {
	if err := m.initialize(); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), m.Timeout)
	defer cancel()

	status, err := m.Client.InstallVersion(ctx, m.OLMNamespace, m.Version)
	if err != nil {
		return err
	}

	log.Infof("Successfully installed OLM version %q", m.Version)
	fmt.Print("\n")
	fmt.Println(status)
	return nil
}

func (m *Manager) Uninstall() error {
	if err := m.initialize(); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), m.Timeout)
	defer cancel()

	if version, err := m.Client.GetInstalledVersion(ctx, m.OLMNamespace); err != nil {
		if m.Version == "" {
			return fmt.Errorf("error getting installed OLM version (set --version to override the default version): %v", err)
		}
	} else if m.Version != "" {
		if version != m.Version {
			return fmt.Errorf("mismatched installed version %q vs. supplied version %q", version, m.Version)
		}
	} else {
		m.Version = version
	}

	if err := m.Client.UninstallVersion(ctx, m.Version); err != nil {
		return err
	}

	log.Infof("Successfully uninstalled OLM version %q", m.Version)
	return nil
}

func (m *Manager) Status() error {
	if err := m.initialize(); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), m.Timeout)
	defer cancel()

	if version, err := m.Client.GetInstalledVersion(ctx, m.OLMNamespace); err != nil {
		if m.Version == "" {
			return fmt.Errorf("error getting installed OLM version (set --version to override the default version): %v", err)
		}
	} else if m.Version != "" {
		if version != m.Version {
			return fmt.Errorf("mismatched installed version %q vs. supplied version %q", version, m.Version)
		}
	} else {
		m.Version = version
	}

	status, err := m.Client.GetStatus(ctx, m.Version)
	if err != nil {
		return err
	}

	log.Infof("Successfully got OLM status for version %q", m.Version)
	fmt.Print("\n")
	fmt.Println(status)
	return nil
}

func (m *Manager) AddToFlagSet(fs *pflag.FlagSet) {
	fs.DurationVar(&m.Timeout, "timeout", DefaultTimeout, "time to wait for the command to complete before failing")
}
