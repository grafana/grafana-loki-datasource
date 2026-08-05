import { toEmitValuesWith } from './toEmitValuesWith';
import { toHaveValueOf } from './toHaveValueOf';
import { Observable } from 'rxjs';

type ObservableType<T> = T extends Observable<infer V> ? V : never;

declare global {
  // eslint-disable-next-line @typescript-eslint/no-namespace
  namespace jest {
    interface Matchers<R, T = {}> {
      toEmitValues<E = ObservableType<T>>(expected: E[]): Promise<CustomMatcherResult>;
      toEmitValuesWith<E = ObservableType<T>>(expectations: (received: E[]) => void): Promise<CustomMatcherResult>;
      toHaveValueOf(expectedValue: unknown): R;
    }
    interface Expect {
      toHaveValueOf(expectedValue: unknown): unknown;
    }
  }
}

export const matchers = {
  toEmitValuesWith,
  toHaveValueOf,
};
