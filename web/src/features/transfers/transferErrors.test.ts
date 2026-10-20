import { ApiError } from '../../api/http';
import { classifyTransferError } from './transferErrors';

const e = (status: number, msg: string) => new ApiError(status, msg);

describe('classifyTransferError', () => {
  it('maps declines by substring after "declined: "', () => {
    expect(classifyTransferError(e(400, 'transaction 1 declined: insufficient funds'))).toEqual({
      kind: 'declined',
      text: 'Недостаточно средств',
    });
    expect(classifyTransferError(e(400, 'transaction 1 declined: denied by antifraud (check 2)')).kind).toBe('declined');
  });

  it('maps field errors', () => {
    expect(classifyTransferError(e(400, 'both accounts must be active')).kind).toBe('field');
    expect(classifyTransferError(e(400, 'currency does not match the account currency')).kind).toBe('field');
    expect(classifyTransferError(e(404, 'account not found')).kind).toBe('field');
    expect(classifyTransferError(e(400, 'to_account_number must be 20 digits')).kind).toBe('field');
  });

  it('maps service errors', () => {
    expect(classifyTransferError(e(503, 'x'))).toMatchObject({ kind: 'fatal', retry: true });
    expect(classifyTransferError(e(500, 'failed to transfer'))).toMatchObject({ kind: 'fatal', retry: true });
    expect(classifyTransferError(e(403, 'x'))).toMatchObject({ kind: 'fatal', retry: false });
  });
});
