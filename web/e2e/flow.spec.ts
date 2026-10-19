import { expect, test, type Page } from '@playwright/test';

// Сквозной сценарий против поднятого стенда: регистрация -> анкета -> документы -> одобрение
// -> открытие счетов -> пополнение -> перевод. Админ - из compose auth-service.
const ADMIN_EMAIL = process.env.E2E_ADMIN_EMAIL || 'admin@banking.local';
const ADMIN_PASSWORD = process.env.E2E_ADMIN_PASSWORD || 'admin123';

const CLIENT_EMAIL = `e2e-${Date.now()}@test.local`;
const CLIENT_PASSWORD = 'E2eTest#2026';

// PNG 1x1: filestore проверяет тип по содержимому, а не по расширению.
const PNG = Buffer.from(
  'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==',
  'base64',
);

async function login(page: Page, email: string, password: string) {
  await page.goto('/login');
  await page.getByLabel('Email').fill(email);
  await page.getByLabel('Пароль').fill(password);
  await page.getByRole('button', { name: 'Войти' }).click();
}

async function logout(page: Page) {
  await page.getByRole('button', { name: 'Выйти' }).first().click();
  await expect(page).toHaveURL(/\/login/);
}

test('client onboarding, officer approval, deposit and transfer', async ({ page }) => {
  // Регистрация и анкета
  await page.goto('/register');
  await page.getByLabel('Email').fill(CLIENT_EMAIL);
  await page.getByLabel('Пароль', { exact: true }).fill(CLIENT_PASSWORD);
  await page.getByLabel('Повторите пароль').fill(CLIENT_PASSWORD);
  await page.getByRole('button', { name: 'Зарегистрироваться' }).click();
  await expect(page).toHaveURL(/\/app\/onboarding/);

  await page.getByLabel('Имя').fill('Ivan');
  await page.getByLabel('Фамилия').fill('Testov');
  await page.getByLabel('Дата рождения').fill('1990-05-17');
  await page.getByLabel('Гражданство').fill('Россия');
  await page.getByLabel('Телефон').fill('+79991234567');
  await page.getByRole('button', { name: 'Сохранить и продолжить' }).click();
  await expect(page.getByText('Шаг 2. Документы')).toBeVisible();

  // Документы: init -> PUT в MinIO -> confirm
  for (const type of ['passport', 'selfie']) {
    await page.getByTestId(`file-${type}`).setInputFiles({ name: `${type}.png`, mimeType: 'image/png', buffer: PNG });
  }
  await expect(page.getByText('Принят')).toHaveCount(2);
  await page.getByRole('button', { name: 'Отправить на проверку' }).click();
  await expect(page.getByText('Шаг 3. Заявка на проверке')).toBeVisible();
  await expect(page.getByText('На рассмотрении').first()).toBeVisible({ timeout: 60_000 });
  await logout(page);

  // Сотрудник берёт и одобряет заявку
  await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
  await expect(page).toHaveURL(/\/staff\/kyc/);
  await page.getByRole('button', { name: 'Взять' }).first().click();
  await expect(page).toHaveURL(/\/staff\/kyc\/.+/);
  await page.getByRole('button', { name: 'Одобрить' }).click();
  await page.getByRole('dialog').getByRole('button', { name: 'Одобрить' }).click();
  await expect(page.getByText('Заявка одобрена')).toBeVisible();
  const userId = await page.getByRole('link', { name: 'Карточка клиента' }).getAttribute('href');
  await logout(page);

  // Клиент открывает два рублёвых счёта
  await login(page, CLIENT_EMAIL, CLIENT_PASSWORD);
  await expect(page).toHaveURL(/\/app$/);
  for (let i = 0; i < 2; i++) {
    await page.getByRole('button', { name: 'Открыть счёт' }).first().click();
    await page.getByRole('dialog').getByRole('button', { name: 'Открыть' }).click();
    await expect(page.getByRole('dialog')).toBeHidden();
  }
  await expect(page.getByText('Активен')).toHaveCount(2);
  await logout(page);

  // Сотрудник пополняет первый счёт
  await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
  await page.goto(`${userId}?tab=accounts`);
  await page.getByText(/^40817 /).first().click();
  await page.getByRole('button', { name: 'Пополнить' }).click();
  await page.getByRole('dialog').getByLabel(/Сумма/).fill('1005,07');
  await page.getByRole('dialog').getByRole('button', { name: 'Пополнить' }).click();
  await expect(page.getByText('Счёт пополнен')).toBeVisible();
  await logout(page);

  // Клиент переводит между своими счетами; двойной клик даёт один запрос
  await login(page, CLIENT_EMAIL, CLIENT_PASSWORD);
  await page.goto('/app/transfer');
  const from = page.getByLabel('Счёт списания');
  const options = await from.locator('option').allTextContents();
  for (const opt of options) {
    await from.selectOption({ label: opt });
    if (await page.getByText('1 005,07').isVisible()) break;
  }
  await page.getByLabel('Счёт зачисления').selectOption({ index: 1 });
  await page.getByLabel('Сумма').fill('100,50');
  await page.getByRole('button', { name: 'Продолжить' }).click();

  let transfers = 0;
  page.on('request', (r) => {
    if (r.method() === 'POST' && r.url().includes('/transfers')) transfers++;
  });
  await page.getByRole('button', { name: 'Подтвердить' }).dblclick();
  await expect(page.getByText('Перевод выполнен')).toBeVisible({ timeout: 60_000 });
  expect(transfers).toBe(1);

  await page.goto('/app');
  await expect(page.getByText('904,57 ₽')).toBeVisible();
  await expect(page.getByText('100,50 ₽').first()).toBeVisible();
});
