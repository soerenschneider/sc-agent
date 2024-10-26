package checkers

import "errors"

func SetMinKsta(ksta int) NeedRestartOpts {
	return func(checker *NeedrestartChecker) error {
		if ksta < 1 || ksta > 2 {
			return errors.New("ksta needs to be [1, 2]")
		}

		checker.rebootMinKsta = ksta
		return nil
	}
}

func SetRebootOnSvc(rebootOnSvc bool) NeedRestartOpts {
	return func(checker *NeedrestartChecker) error {
		checker.rebootOnSvc = rebootOnSvc
		return nil
	}
}

func NeedrestartCheckerFromMap(args map[string]any) (*NeedrestartChecker, error) {
	if args == nil {
		return NewNeedrestartChecker()
	}

	var opts []NeedRestartOpts
	rebootOnSvc, ok := args["reboot_on_svc"].(bool)
	if ok {
		opts = append(opts, SetRebootOnSvc(rebootOnSvc))
	}

	minKstaVal, ok := args["min_ksta"].(float64)
	if ok {
		value := int(minKstaVal)
		opts = append(opts, SetMinKsta(value))
	}

	return NewNeedrestartChecker(opts...)
}
