
---
title: "贡献"
linkTitle: "贡献指南"
weight: 159
description: >
  了解如何为 EdgeVPN 做出贡献
---

## 为 EdgeVPN 做贡献

EdgeVPN 项目的贡献指南位于 [GitHub 仓库](https://github.com/mudler/edgevpn/blob/master/CONTRIBUTING.md)。这里是为文档网站贡献的一些提示。

## 为文档网站做贡献

### 我们使用 GitHub 进行开发

我们使用 [GitHub 托管代码](https://github.com/mudler/edgevpn)、跟踪问题和功能请求，以及接受 Pull Request。

我们使用 [Hugo](https://gohugo.io/) 来格式化和生成网站，使用 [Docsy](https://github.com/google/docsy) 主题进行样式设计和网站结构，使用 GitHub Actions 管理网站部署。Hugo 是一个开源的静态网站生成器，为我们提供模板、标准目录结构中的内容组织以及网站生成引擎。你可以用 Markdown（或者 HTML，如果你愿意）编写页面，Hugo 会将它们打包成一个网站。

所有提交，包括项目成员的提交，都需要审查。我们使用 GitHub pull request 进行此目的。有关使用 pull request 的更多信息，请参阅 [GitHub 帮助](https://help.github.com/articles/about-pull-requests/)。

### 你所做的任何贡献都将受仓库软件许可证的约束

简而言之，当你提交代码更改时，你的提交将被理解为受覆盖项目的相同许可证的约束。如果这是一个问题，请随时联系维护者。

### 更新单个页面

如果你在使用文档时发现了一些想要更改的内容，Docsy 为你提供了一个快捷方式：

1. 在你想要修改的页面右上角点击 **Edit this page**。
2. 如果你还没有项目仓库的最新 fork，系统会提示你获取一个 - 点击 **Fork this repository and propose changes** 或 **Update your Fork** 以获取项目的最新版本进行编辑。你的 fork 中的相应页面将以编辑模式显示。


### 使用本地检出快速开始

这是使用 git 本地检出更新文档的快速指南。它假设你熟悉 GitHub 工作流程，并且愿意使用文档更新的自动预览：

1. 在 GitHub 上 Fork [仓库](https://github.com/mudler/edgevpn)。
2. 进行更改，如果与文档相关，要查看预览，请在 `docs` 目录中运行 `make serve`，然后浏览到 [localhost:1313](http://localhost:1313)
3. 如果你还没有准备好进行审查，请在 PR 名称中添加 "WIP" 以表示这是一个正在进行的工作。
4. 继续更新文档并推送更改，直到你对内容满意为止。
5. 当你准备好进行审查时，在 PR 中添加评论，并删除任何 "WIP" 标记。
6. 当你满意时，发送 pull request (PR)。

### 许可证

通过贡献，你同意你的贡献将根据项目许可证进行许可。
