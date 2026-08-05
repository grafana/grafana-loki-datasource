import { matcherHint, printExpected, printReceived } from 'jest-matcher-utils';

/**
 * Passes when `received.valueOf()` strictly equals `expectedValue`.
 *
 * This compares date-like objects through their public contract instead of relying on Moment internals.
 */
export function toHaveValueOf(received: { valueOf(): unknown }, expectedValue: unknown): jest.CustomMatcherResult {
  const receivedValue = received.valueOf();
  const pass = receivedValue === expectedValue;

  return {
    pass,
    message: () =>
      pass
        ? `${matcherHint('.not.toHaveValueOf')}

  Expected valueOf() ${printReceived(receivedValue)} not to equal ${printExpected(expectedValue)}`
        : `${matcherHint('.toHaveValueOf')}

  Expected valueOf() ${printReceived(receivedValue)} to equal ${printExpected(expectedValue)}`,
  };
}
