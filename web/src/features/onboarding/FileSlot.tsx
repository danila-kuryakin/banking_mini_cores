import { useQueryClient } from '@tanstack/react-query';
import clsx from 'clsx';
import { CheckCircle2, ExternalLink, FileText, ImageIcon, UploadCloud, XCircle } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import { errorText } from '../../api/errors';
import { putToStorage } from '../../api/http';
import { filesApi } from '../../api/files';
import type { FileRecord } from '../../api/types';
import { Badge } from '../../components/Badge';
import { Button } from '../../components/Button';
import { ErrorDetails } from '../../components/Feedback';
import { FILE_TYPE_LABELS } from '../../lib/statuses';
import { FILE_ACCEPT, fileError } from '../../lib/validation';
import { qk } from '../client/queries';

type Phase = 'idle' | 'uploading' | 'checking' | 'error';

export function openFile(userId: string, fileId: string) {
  // Окно открываем синхронно, иначе браузер заблокирует его как всплывающее.
  const win = window.open('', '_blank');
  void filesApi
    .downloadUrl(userId, fileId)
    .then((t) => {
      if (win) win.location.href = t.url;
      else window.open(t.url, '_blank');
    })
    .catch(() => win?.close());
}

// Слот документа: init -> PUT в MinIO по presigned-ссылке -> confirm, всё за одно действие.
export function FileSlot({
  type,
  confirmed,
  required,
  requested,
}: {
  type: string;
  confirmed?: FileRecord;
  required: boolean;
  requested?: boolean;
}) {
  const queryClient = useQueryClient();
  const inputRef = useRef<HTMLInputElement>(null);
  const [phase, setPhase] = useState<Phase>('idle');
  const [progress, setProgress] = useState(0);
  const [error, setError] = useState<{ text: string; raw?: unknown } | null>(null);
  const [preview, setPreview] = useState<string | null>(null);
  const [drag, setDrag] = useState(false);

  useEffect(() => () => void (preview && URL.revokeObjectURL(preview)), [preview]);

  const upload = async (file: File) => {
    const invalid = fileError(file);
    if (invalid) {
      setPhase('error');
      setError({ text: invalid });
      return;
    }
    setError(null);
    setPreview(file.type.startsWith('image/') ? URL.createObjectURL(file) : null);
    setPhase('uploading');
    setProgress(0);
    try {
      const ticket = await filesApi.init('me', type, file.name);
      await putToStorage(ticket.upload_url, file, setProgress);
      setPhase('checking');
      await filesApi.confirm('me', ticket.file_id);
      setPhase('idle');
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: qk.files('me') }),
        queryClient.invalidateQueries({ queryKey: qk.required('me') }),
      ]);
    } catch (err) {
      setPhase('error');
      setError({
        raw: err,
        text: errorText(err, {
          400: 'Файл повреждён или не jpeg/png/pdf',
          409: 'Документ этого типа уже принят',
          0: 'Не удалось загрузить файл в хранилище',
        }),
      });
    }
  };

  const busy = phase === 'uploading' || phase === 'checking';
  const status = confirmed
    ? { label: 'Принят', tone: 'up' as const }
    : phase === 'uploading'
      ? { label: 'Загружается', tone: 'info' as const }
      : phase === 'checking'
        ? { label: 'Проверяется', tone: 'warn' as const }
        : phase === 'error'
          ? { label: 'Ошибка', tone: 'down' as const }
          : { label: 'Не загружен', tone: 'muted' as const };

  return (
    <div
      className={clsx(
        'card flex flex-col border-2 border-dashed p-4 transition-colors',
        drag ? 'border-accent' : confirmed ? 'border-up/40 border-solid' : 'border-border',
      )}
      onDragOver={(e) => {
        if (confirmed || busy) return;
        e.preventDefault();
        setDrag(true);
      }}
      onDragLeave={() => setDrag(false)}
      onDrop={(e) => {
        e.preventDefault();
        setDrag(false);
        const f = e.dataTransfer.files[0];
        if (f && !confirmed && !busy) void upload(f);
      }}
    >
      <div className="flex items-start justify-between gap-2">
        <div>
          <div className="font-medium">{FILE_TYPE_LABELS[type] ?? type}</div>
          <div className="text-xs text-muted">
            {requested ? 'Запрошен сотрудником' : required ? 'Обязательный' : 'Необязательный'}
          </div>
        </div>
        <Badge tone={status.tone}>{status.label}</Badge>
      </div>

      <div className="my-4 flex h-32 items-center justify-center overflow-hidden rounded-lg bg-input">
        {preview ? (
          <img src={preview} alt="" className="h-full w-full object-cover" />
        ) : confirmed ? (
          confirmed.content_type === 'application/pdf' ? (
            <FileText size={40} className="text-up" strokeWidth={1.5} />
          ) : (
            <ImageIcon size={40} className="text-up" strokeWidth={1.5} />
          )
        ) : (
          <UploadCloud size={40} className="text-muted" strokeWidth={1.5} />
        )}
      </div>

      {busy && (
        <div className="mb-3">
          <div className="h-1.5 overflow-hidden rounded-full bg-input">
            <div
              className={clsx('h-full bg-accent transition-all', phase === 'checking' && 'animate-pulse')}
              style={{ width: `${phase === 'checking' ? 100 : progress}%` }}
            />
          </div>
          <div className="mt-1 text-xs text-muted num">{phase === 'checking' ? 'Проверяем файл…' : `${progress}%`}</div>
        </div>
      )}

      {error && (
        <div className="mb-3 flex items-start gap-2 text-xs text-down">
          <XCircle size={14} className="mt-0.5 shrink-0" />
          <div>
            {error.text}
            {!!error.raw && <ErrorDetails error={error.raw} />}
          </div>
        </div>
      )}

      <div className="mt-auto">
        {confirmed ? (
          <div className="flex items-center justify-between gap-2">
            <span className="flex min-w-0 items-center gap-1.5 text-xs text-muted">
              <CheckCircle2 size={14} className="shrink-0 text-up" />
              <span className="truncate">{confirmed.filename}</span>
            </span>
            <Button size="sm" variant="ghost" onClick={() => openFile('me', confirmed.id)}>
              <ExternalLink size={14} /> Открыть
            </Button>
          </div>
        ) : (
          <>
            <input
              ref={inputRef}
              type="file"
              accept={FILE_ACCEPT}
              className="hidden"
              data-testid={`file-${type}`}
              onChange={(e) => {
                const f = e.target.files?.[0];
                e.target.value = '';
                if (f) void upload(f);
              }}
            />
            <Button size="sm" variant="secondary" block loading={busy} onClick={() => inputRef.current?.click()}>
              Выбрать файл
            </Button>
            <p className="mt-2 text-center text-[11px] text-muted">или перетащите сюда · jpg, png, pdf до 10 МБ</p>
          </>
        )}
      </div>
    </div>
  );
}
