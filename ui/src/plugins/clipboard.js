export default {
  clipboard: navigator.clipboard,
  clipboardData: window.clipboardData,

  write(text) {
    this.clipboard.writeText(text).then();
    this.clipboardData.setData('Text', text);
  },

  read() {
    if (this.clipboard)
      return this.clipboard.readText();

    if (this.clipboardData)
      return new Promise<string>(resolve => resolve(this.clipboardData?.getData('Text') || ''));

    return new Promise<string>(resolve => resolve(''));
  }
}
