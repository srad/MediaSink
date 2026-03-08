# MediaSink.Vue

![License](https://img.shields.io/badge/license-AGPL--v3-blue)
![Go Version](https://img.shields.io/badge/Go-1.x-blue)
[![Build Status](https://teamcity.sedrad.com/app/rest/builds/buildType:(id:MediaSinkVue_Build)/statusIcon)](https://teamcity.sedrad.com/viewType.html?buildTypeId=MediaSinkVue_Build&guest=1)
![Build](https://img.shields.io/github/actions/workflow/status/srad/MediaSink.Vue/build.yml)
 
## Project Setup

```sh
pnpm install
pnpm approve-builds # for pnpm 10+
```

### Compile and Hot-Reload for Development

```sh
pnpm dev
```

### Type-Check, Compile and Minify for Production

```sh
pnpm build
```

### Run Unit Tests with [Vitest](https://vitest.dev/)

```sh
pnpm test:unit
```

### Run End-to-End Tests with [Nightwatch](https://nightwatchjs.org/)

```sh
# When using CI, the project must be built first.
pnpm build

# Runs the end-to-end tests
pnpm test:e2e
# Runs the tests only on Chrome
pnpm test:e2e --env chrome
# Runs the tests of a specific file
pnpm test:e2e tests/e2e/example.ts
# Runs the tests in debug mode
pnpm test:e2e --debug
```
    
### Lint with [ESLint](https://eslint.org/)

```sh
pnpm lint
```

## Contributing
We welcome contributions! To get started:
1. Fork the repository.
2. Create a new branch.
3. Make your changes and commit them.
4. Submit a pull request.

## License
MediaSink.Go is dual-licensed under the GNU Affero General Public License (AGPL) and a commercial license.

- **Open-Source Use (AGPL License)**: MediaSink.Go is free to use, modify, and distribute under the terms of the [GNU AGPL v3](https://www.gnu.org/licenses/agpl-3.0.html). Any modifications and derivative works must also be open-sourced under the same license.
- **Commercial Use**: Companies that wish to use MediaSink.Go without AGPL restrictions must obtain a commercial license. For more details, please refer to the [LICENSE](LICENSE) file or contact us for licensing inquiries.
MediaSink.Go is available for free for non-profit and educational institutions. However, a commercial license is required for companies. For more details, please refer to the [LICENSE](LICENSE) file or contact us for licensing inquiries.
This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Contact
For issues and feature requests, please use the [GitHub Issues](https://github.com/srad/MediaSink.Vue/issues) section.
