import { birthDateError, isCalendarDate, profileSchema } from './validation';

const empty = { first_name: '', last_name: '', birth_date: '', citizenship: '', phone: '' };

describe('isCalendarDate', () => {
  it('rejects non-calendar dates', () => {
    expect(isCalendarDate('2023-02-29')).toBe(false);
    expect(isCalendarDate('2024-02-29')).toBe(true);
    expect(isCalendarDate('2024-04-31')).toBe(false);
    expect(isCalendarDate('24-01-01')).toBe(false);
  });
});

describe('birthDateError', () => {
  const now = new Date(2026, 9, 19);
  it('accepts empty and valid', () => {
    expect(birthDateError('', now)).toBeNull();
    expect(birthDateError('1990-05-17', now)).toBeNull();
  });
  it('rejects future and too old', () => {
    expect(birthDateError('2030-01-01', now)).not.toBeNull();
    expect(birthDateError('1800-01-01', now)).not.toBeNull();
  });
});

describe('profileSchema', () => {
  it('trims and allows partial', () => {
    const r = profileSchema.parse({ ...empty, first_name: '  Ivan ' });
    expect(r.first_name).toBe('Ivan');
  });
  it('checks phone', () => {
    expect(profileSchema.safeParse({ ...empty, phone: '12' }).success).toBe(false);
    expect(profileSchema.safeParse({ ...empty, phone: '+79991234567' }).success).toBe(true);
  });
});
