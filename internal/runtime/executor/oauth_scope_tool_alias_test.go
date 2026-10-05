package executor

import (
	"fmt"
	"sync"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v8/internal/config"
)

func TestAPIKeyScopeSharesClaudeToolAliasStore(t *testing.T) {
	for _, provider := range []string{"claude", "kimi"} {
		t.Run(provider, func(t *testing.T) {
			cfg := &config.Config{}
			var original *ClaudeExecutor
			var scope func() *ClaudeExecutor
			if provider == "claude" {
				executor := NewClaudeExecutor(cfg)
				original = executor
				scope = func() *ClaudeExecutor {
					return executor.ForAPIKey().(*ClaudeExecutor)
				}
			} else {
				executor := NewKimiExecutor(cfg)
				original = &executor.ClaudeExecutor
				scope = func() *ClaudeExecutor {
					return &executor.ForAPIKey().(*KimiExecutor).ClaudeExecutor
				}
			}

			const workers = 16
			views := make(chan *ClaudeExecutor, workers)
			var pending sync.WaitGroup
			for i := range workers {
				pending.Add(1)
				go func() {
					defer pending.Done()
					view := scope()
					key := fmt.Sprintf("message:%d", i)
					view.claudeOAuthToolAliasStore().save(
						[]string{key}, map[string]string{"wire_name": "client_name"},
					)
					views <- view
				}()
			}
			pending.Wait()
			close(views)
			store := original.claudeOAuthToolAliasStore()
			for view := range views {
				if view == original || view.cfg != cfg.ForAPIKey() {
					t.Fatal("scope must create a separate executor with scoped config")
				}
				if view.claudeOAuthToolAliasStore() != store {
					t.Fatal("scope must retain the shared tool alias store")
				}
				if view.requestLogProvider != original.requestLogProvider {
					t.Fatal("scope changed the request log provider")
				}
			}
			for i := range workers {
				key := fmt.Sprintf("message:%d", i)
				aliases, ok := store.load([]string{key})
				if !ok || aliases["wire_name"] != "client_name" {
					t.Fatalf("scope lost continuation aliases for %s", key)
				}
			}
			if original.cfg != cfg {
				t.Fatal("scope changed the original executor config")
			}
		})
	}
}
