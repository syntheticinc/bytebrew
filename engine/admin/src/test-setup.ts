import '@testing-library/jest-dom';

// jsdom does not implement HTMLDialogElement methods. Modal uses showModal()/close()
// via a ref; polyfill them here so modal-opening tests don't crash.
if (typeof HTMLDialogElement !== 'undefined') {
  const proto = HTMLDialogElement.prototype as unknown as {
    showModal?: () => void;
    close?: () => void;
    show?: () => void;
  };
  if (!proto.showModal) {
    proto.showModal = function () {
      (this as unknown as HTMLDialogElement).setAttribute('open', '');
    };
  }
  if (!proto.close) {
    proto.close = function () {
      (this as unknown as HTMLDialogElement).removeAttribute('open');
    };
  }
  if (!proto.show) {
    proto.show = function () {
      (this as unknown as HTMLDialogElement).setAttribute('open', '');
    };
  }
}
