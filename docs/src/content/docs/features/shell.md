---
title: Shell Completions
description: Learn how to use Doggo's shell completions for zsh and fish
---

Doggo provides shell completions for `zsh` and `fish`, enhancing your command-line experience with auto-completion for commands and options.

### Bash Completions

To enable Bash completions for Doggo:

1. Generate the completion script:

   ```bash
   doggo completions bash > doggo_completion.bash
   ```

2. Source the generated file in your `.bashrc` or `.bash_profile`:

   ```bash
   echo "source ~/path/to/doggo_completion.bash" >> ~/.bashrc
   ```

   Replace `~/path/to/` with the actual path where you saved the completion script.

3. Restart your shell or run `source ~/.bashrc`.


### Zsh Completions

To enable Zsh completions for Doggo:

1. Ensure that shell completions are enabled in your `.zshrc` file:

   ```zsh
   autoload -U compinit
   compinit
   ```

2. Generate the completion script and add it to your Zsh functions path:

   ```bash
   doggo completions zsh > "${fpath[1]}/_doggo"
   ```

3. Restart your shell or run `source ~/.zshrc`.

Now you can use Tab to auto-complete Doggo commands and options in Zsh.

### Fish Completions

To enable Fish completions for Doggo:

1. Generate the completion script and save it to the Fish completions directory:

   ```bash
   $ doggo completions fish > ~/.config/fish/completions/doggo.fish
   ```

2. Restart your Fish shell or run `source ~/.config/fish/config.fish`.

You can now use Tab to auto-complete Doggo commands and options in Fish.

### Using Completions

With completions enabled, you can:

- Auto-complete command-line flags (e.g., `doggo --<Tab>`)
- Auto-complete DNS record types (e.g., `doggo -t <Tab>`)
- Auto-complete subcommands and options
