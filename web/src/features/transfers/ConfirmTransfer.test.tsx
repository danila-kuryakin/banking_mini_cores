import { fireEvent, render, screen } from '@testing-library/react';
import { ConfirmTransfer } from './ConfirmTransfer';

const draft = { fromLabel: 'A', toLabel: 'B', amount: 1000, currency: 'RUB' };

describe('ConfirmTransfer', () => {
  it('sends one request on double click', () => {
    const onConfirm = vi.fn();
    render(<ConfirmTransfer draft={draft} pending={false} onConfirm={onConfirm} onBack={() => {}} />);
    const button = screen.getByRole('button', { name: 'Подтвердить' });
    fireEvent.click(button);
    fireEvent.click(button);
    fireEvent.doubleClick(button);
    expect(onConfirm).toHaveBeenCalledTimes(1);
  });

  it('is disabled while pending', () => {
    render(<ConfirmTransfer draft={draft} pending onConfirm={() => {}} onBack={() => {}} />);
    expect(screen.getByRole('button', { name: /Загрузка/ })).toBeDisabled();
  });
});
