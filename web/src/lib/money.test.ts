import { formatAccountNumber, formatMoney, fromMinor, toMinor } from './money';

describe('toMinor', () => {
  it.each([
    ['0', 0],
    ['1', 100],
    ['1.5', 150],
    ['1,5', 150],
    ['0.01', 1],
    ['0.10', 10],
    ['19.99', 1999],
    ['1 234,56', 123456],
    ['1.', 100],
    ['0.29', 29],
    ['4.35', 435],
    ['1005.07', 100507],
  ])('%s -> %d', (input, expected) => {
    expect(toMinor(input)).toBe(expected);
  });

  it.each(['', 'abc', '1.234', '-1', '1.2.3', '.5', '1e3'])('rejects %s', (input) => {
    expect(toMinor(input)).toBeNull();
  });

  it('keeps every kopeck on round trip', () => {
    for (let i = 0; i < 100_000; i += 7) {
      expect(toMinor(fromMinor(i))).toBe(i);
    }
  });
});

describe('fromMinor', () => {
  it('formats with two digits', () => {
    expect(fromMinor(0)).toBe('0.00');
    expect(fromMinor(5)).toBe('0.05');
    expect(fromMinor(123456)).toBe('1234.56');
    expect(fromMinor(-150)).toBe('-1.50');
  });
});

describe('formatMoney', () => {
  it('does not lose kopecks', () => {
    const s = formatMoney(100507, 'RUB').replace(/[\s\u00a0\u202f]/g, '');
    expect(s).toBe('1005,07₽');
    expect(formatMoney(29, 'USD')).toContain('0,29');
    expect(formatMoney(1, 'EUR')).toContain('0,01');
  });

  it('keeps kopecks for large amounts', () => {
    const s = formatMoney(99_999_999_999_99, 'RUB').replace(/[\s\u00a0\u202f]/g, '');
    expect(s).toBe('99999999999,99₽');
  });
});

describe('formatAccountNumber', () => {
  it('groups 20 digits', () => {
    expect(formatAccountNumber('40817810500000000001')).toBe('40817 810 5 0000 0000001');
    expect(formatAccountNumber('123')).toBe('123');
  });
});
