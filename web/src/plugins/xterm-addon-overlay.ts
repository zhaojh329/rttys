import {Terminal, ITerminalAddon} from 'xterm'

export class OverlayAddon implements ITerminalAddon {
  private terminal: Terminal | undefined;
  private overlayNode: HTMLDivElement;
  private timeout: NodeJS.Timeout | undefined;

  constructor() {
    this.overlayNode = document.createElement('div');
    this.overlayNode.style.cssText = `
      border-radius: 15px;
      font-size: xx-large;
      opacity: 0.75;
      padding: 0.2em 0.5em 0.2em 0.5em;
      position: absolute;
      -webkit-user-select: none;
      -webkit-transition: opacity 180ms ease-in;
      -moz-user-select: none;
      -moz-transition: opacity 180ms ease-in;`;
    this.overlayNode.style.color = '#101010';
    this.overlayNode.style.backgroundColor = '#f0f0f0';
    this.overlayNode.style.opacity = '0.75';
  }

  public activate(terminal: Terminal): void {
    this.terminal = terminal;
  }

  public dispose(): void {
    return;
  }

  public show(msg: string, timeout?: number): void {
    const {terminal, overlayNode} = this;

    if (!terminal || !terminal.element)
      return;

    overlayNode.style.opacity = '0.75';
    overlayNode.textContent = msg;

    if (!overlayNode.parentNode)
      terminal.element.appendChild(overlayNode);

    const divSize = terminal.element.getBoundingClientRect();
    const overlaySize = overlayNode.getBoundingClientRect();

    overlayNode.style.top = (divSize.height - overlaySize.height) / 2 + 'px';
    overlayNode.style.left = (divSize.width - overlaySize.width) / 2 + 'px';

    if (this.timeout)
      clearTimeout(this.timeout);

    this.timeout = setTimeout(() => {
      overlayNode.style.opacity = '0';

      this.timeout = setTimeout(() => {
        if (overlayNode.parentNode)
          overlayNode.parentNode.removeChild(overlayNode);

        overlayNode.style.opacity = '0.75';
        this.timeout = undefined;
      }, 200);
    }, timeout || 1500);
  }
}
