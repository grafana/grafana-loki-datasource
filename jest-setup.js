import React from 'react';
import './.config/jest-setup';
import { matchers } from './src/test/matchers';
import { MessageChannel } from 'worker_threads';

// Make MessageChannel available globally for jsdom (needed by @rc-component/select)
if (!global.MessageChannel) {
  global.MessageChannel = MessageChannel;
}

global.React = React;

const mockIntersectionObserver = jest.fn().mockImplementation((callback) => ({
  observe: jest.fn().mockImplementation((elem) => {
    callback([{ target: elem, isIntersecting: true }]);
  }),
  unobserve: jest.fn(),
  disconnect: jest.fn(),
}));
global.IntersectionObserver = mockIntersectionObserver;

// @grafana/ui's VirtualizedSelectMenu (and other components) use ResizeObserver to
// size their FixedSizeList. A no-op mock causes those lists to render no items.
// We fire the callback synchronously on observe() so dimensions are available
// immediately — avoiding the timing race that caused 5000ms CI timeouts when
// using setTimeout (which doesn't fire reliably before assertions on Linux/Node 22).
global.ResizeObserver = class ResizeObserver {
  static _observationEntry = {
    contentRect: {
      x: 1,
      y: 2,
      width: 500,
      height: 500,
      top: 100,
      bottom: 0,
      left: 100,
      right: 0,
    },
    target: {
      getAttribute: () => 1,
    },
  };

  _isObserving = false;
  _callback;

  constructor(callback) {
    this._callback = callback;
  }

  observe() {
    this._isObserving = true;
    this._callback([ResizeObserver._observationEntry], this);
  }

  disconnect() {
    this._isObserving = false;
  }

  unobserve() {
    this._isObserving = false;
  }
};

// Mock document.fonts for Monaco editor
Object.defineProperty(document, 'fonts', {
  value: {
    ready: Promise.resolve(),
    load: jest.fn(() => Promise.resolve([])),
    check: jest.fn(() => true),
  },
  writable: true,
});

// Mock canvas getContext for Monaco editor text measurement
HTMLCanvasElement.prototype.getContext = jest.fn(() => ({
  measureText: jest.fn(() => ({ width: 0 })),
  fillText: jest.fn(),
  clearRect: jest.fn(),
  fillRect: jest.fn(),
  beginPath: jest.fn(),
  moveTo: jest.fn(),
  lineTo: jest.fn(),
  stroke: jest.fn(),
  save: jest.fn(),
  restore: jest.fn(),
  scale: jest.fn(),
  translate: jest.fn(),
  drawImage: jest.fn(),
}));

expect.extend(matchers);

// jsdom doesn't compute CSS display values from user-agent stylesheets.
// dom-accessibility-api uses getComputedStyle(el).getPropertyValue('display') to
// decide whether to insert a space separator between text nodes when computing
// accessible names. Without this patch, inline elements like <mark> and <span>
// (which PartialHighlighter from @grafana/ui uses) get treated as block-level,
// splitting "value1-1" into "val  ue  1  -1" and breaking accessible name queries.
const _origGetComputedStyle = window.getComputedStyle.bind(window);
const _inlineElements = new Set([
  'A', 'ABBR', 'ACRONYM', 'B', 'BDO', 'BIG', 'BR', 'BUTTON', 'CITE', 'CODE',
  'DFN', 'EM', 'I', 'IMG', 'INPUT', 'KBD', 'LABEL', 'MAP', 'MARK', 'OUTPUT',
  'Q', 'SAMP', 'SELECT', 'SMALL', 'SPAN', 'STRONG', 'S', 'SUB', 'SUP',
  'TEXTAREA', 'TIME', 'TT', 'U', 'VAR',
]);
Object.defineProperty(window, 'getComputedStyle', {
  value: (element, pseudo) => {
    const style = _origGetComputedStyle(element, pseudo);
    if (!pseudo && element && element.tagName && _inlineElements.has(element.tagName)) {
      const existingDisplay = element.style && element.style.display;
      if (!existingDisplay) {
        return new Proxy(style, {
          get(target, prop) {
            if (prop === 'display') {
              return 'inline';
            }
            if (prop === 'getPropertyValue') {
              return (name) => (name === 'display' ? 'inline' : target.getPropertyValue(name));
            }
            const val = target[prop];
            return typeof val === 'function' ? val.bind(target) : val;
          },
        });
      }
    }
    return style;
  },
  writable: true,
  configurable: true,
});
