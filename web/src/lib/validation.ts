import { z } from 'zod';

// Правила дублируют бэкенд (гейтвей, customer-service, kyc-service), чтобы ошибки ловились до запроса.
export const email = z.string().trim().min(1, 'Укажите email').email('Неверный формат email');
export const password = z.string().min(8, 'Минимум 8 символов').max(72, 'Максимум 72 символа');

const PHONE_RE = /^\+?[0-9]{7,15}$/;

export function isCalendarDate(value: string): boolean {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value);
  if (!m) return false;
  const [y, mo, d] = [Number(m[1]), Number(m[2]), Number(m[3])];
  const dt = new Date(Date.UTC(y, mo - 1, d));
  return dt.getUTCFullYear() === y && dt.getUTCMonth() === mo - 1 && dt.getUTCDate() === d;
}

export function ageOn(birth: string, now = new Date()): number {
  const [y, m, d] = birth.split('-').map(Number) as [number, number, number];
  let age = now.getFullYear() - y;
  if (now.getMonth() + 1 < m || (now.getMonth() + 1 === m && now.getDate() < d)) age--;
  return age;
}

export function birthDateError(value: string, now = new Date()): string | null {
  if (!value) return null;
  if (!isCalendarDate(value)) return 'Несуществующая дата';
  const [y, m, d] = value.split('-').map(Number) as [number, number, number];
  if (new Date(y, m - 1, d) > now) return 'Дата в будущем';
  if (ageOn(value, now) > 200) return 'Слишком давняя дата';
  return null;
}

// Анкету можно сохранять частично, поэтому каждое поле может быть пустым.
const optionalText = (max: number) =>
  z
    .string()
    .transform((s) => s.trim())
    .refine((s) => s.length <= max, `Не длиннее ${max} символов`);

export const profileSchema = z.object({
  first_name: optionalText(100),
  last_name: optionalText(100),
  birth_date: z.string().superRefine((v, ctx) => {
    const err = birthDateError(v);
    if (err) ctx.addIssue({ code: z.ZodIssueCode.custom, message: err });
  }),
  citizenship: optionalText(64),
  phone: z
    .string()
    .transform((s) => s.trim())
    .refine((s) => s === '' || PHONE_RE.test(s), 'Формат: +79991234567, 7-15 цифр'),
});

export type ProfileFormInput = z.input<typeof profileSchema>;
export type ProfileFormOutput = z.output<typeof profileSchema>;

export const reason = z.string().trim().min(1, 'Укажите причину').max(500, 'Не длиннее 500 символов');

export const MAX_FILE_SIZE = 10 * 1024 * 1024;
const ALLOWED_EXT = ['jpg', 'jpeg', 'png', 'pdf'];
export const FILE_ACCEPT = '.jpg,.jpeg,.png,.pdf,image/jpeg,image/png,application/pdf';

export function fileError(file: File): string | null {
  const ext = file.name.split('.').pop()?.toLowerCase() ?? '';
  if (!ALLOWED_EXT.includes(ext)) return 'Только jpg, png или pdf';
  if (file.size > MAX_FILE_SIZE) return 'Файл больше 10 МБ';
  if (file.size === 0) return 'Пустой файл';
  return null;
}

export function passwordStrength(p: string): 0 | 1 | 2 | 3 | 4 {
  let score = 0;
  if (p.length >= 8) score++;
  if (p.length >= 12) score++;
  if (/[a-zа-я]/.test(p) && /[A-ZА-Я]/.test(p)) score++;
  if (/\d/.test(p) && /[^\p{L}\d]/u.test(p)) score++;
  return Math.min(score, 4) as 0 | 1 | 2 | 3 | 4;
}
