# Git Setup in Dev Containers

This project runs in a Dev Container. To ensure your commits are properly attributed, you need to have `user.name` and `user.email` configured in Git.

There are two recommended ways to handle this automatically:

## 1. Host Git Configuration (Recommended)
VS Code can automatically copy your local Git configuration into the container.

1.  Ensure you have Git installed on your host machine (Windows/Mac/Linux).
2.  Configure your identity globally on your host:
    ```bash
    git config --global user.name "Your Name"
    git config --global user.email "your.email@example.com"
    ```
3.  VS Code Dev Containers will automatically detect this and forward it to the container when you rebuild or reopen it.

## 2. Dotfiles
If you maintain a dotfiles repository, you can configure VS Code to clone and install your dotfiles in the container.

1.  Open VS Code Settings (`Cmd+,` or `Ctrl+,`).
2.  Search for "Dotfiles".
3.  Set `Dotfiles: Repository` to your dotfiles repo URL.
4.  Standard `.gitconfig` files in your home directory will be respected.

## Manual Setup (Temporary)
If you just want to set it for this session:

```bash
git config user.name "Your Name"
git config user.email "your.email@example.com"
```

*Note: This will be lost if you rebuild the container.*
