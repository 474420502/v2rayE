package native

import (
	xraycore "github.com/xtls/xray-core/core"
	_ "github.com/xtls/xray-core/main/distro/all"
	_ "github.com/xtls/xray-core/main/json"
)

type managedXrayCore struct {
	instance       *xraycore.Instance
	restoreLogFunc func() // restores the previous xray-core log handler on close
}

func startManagedXrayCore(configJSON []byte, broker *logBroker) (*managedXrayCore, error) {
	// Register our log handler BEFORE starting the instance so we capture
	// all startup output in embedded mode.
	restore := RegisterXrayLogHandler(broker)

	instance, err := xraycore.StartInstance("json", configJSON)
	if err != nil {
		restore() // revert handler if start failed
		return nil, err
	}
	return &managedXrayCore{instance: instance, restoreLogFunc: restore}, nil
}

func (c *managedXrayCore) Close() error {
	if c == nil || c.instance == nil {
		return nil
	}
	err := c.instance.Close()
	if c.restoreLogFunc != nil {
		c.restoreLogFunc()
	}
	return err
}

func (c *managedXrayCore) IsRunning() bool {
	if c == nil || c.instance == nil {
		return false
	}
	return c.instance.IsRunning()
}

