'use strict';

Object.defineProperty(exports, '__esModule', { value: true });

var withInstall = require('../../utils/with-install.js');
var index = require('./src/index.js');
var configProvider = require('./src/config-provider.js');

const ElConfigProvider = withInstall.withInstall(index.ConfigProvider);

exports.configProviderProps = configProvider.configProviderProps;
exports.ElConfigProvider = ElConfigProvider;
exports["default"] = ElConfigProvider;
//# sourceMappingURL=index.js.map
