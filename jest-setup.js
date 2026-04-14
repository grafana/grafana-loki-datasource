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

global.ResizeObserver = jest.fn().mockImplementation(() => ({
  observe: jest.fn(),
  unobserve: jest.fn(),
  disconnect: jest.fn(),
}));

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
