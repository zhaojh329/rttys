interface ClipboardDataIE {
  getData(format: string): string;
  setData(format: string, data: string): void;
}

class ClipboardEx {
  private readonly clipboard: Clipboard | undefined;
  private readonly clipboardData: ClipboardDataIE | undefined;

  constructor() {
    this.clipboardData = (window as any).clipboardData;
    this.clipboard = navigator.clipboard;
  }

  write(text: string) {
      this.clipboard?.writeText(text).then();
      this.clipboardData?.setData('Text', text);
  }

  read(): Promise<string> {
    if (this.clipboard)
      return this.clipboard.readText();

    if (this.clipboardData)
      return new Promise<string>(resolve => resolve(this.clipboardData?.getData('Text')));

    return new Promise<string>(resolve => resolve(''));
  }
}

export default new ClipboardEx()
