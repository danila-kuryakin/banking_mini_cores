import { useQuery } from '@tanstack/react-query';
import { ExternalLink, FileText, FolderOpen } from 'lucide-react';
import { useState } from 'react';
import { filesApi } from '../../api/files';
import type { FileRecord } from '../../api/types';
import { StatusBadge } from '../../components/Badge';
import { Button } from '../../components/Button';
import { EmptyState } from '../../components/Feedback';
import { Modal } from '../../components/Modal';
import { Skeleton } from '../../components/Spinner';
import { Table, type Column } from '../../components/Table';
import { formatDateTime } from '../../lib/dates';
import { FILE_TYPE_LABELS } from '../../lib/statuses';
import { openFile } from '../onboarding/FileSlot';

function size(bytes: number): string {
  if (!bytes) return '—';
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)} КБ`;
  return `${(bytes / 1024 / 1024).toFixed(1)} МБ`;
}

export function DocumentsList({
  userId,
  files,
  loading,
}: {
  userId: string;
  files: FileRecord[] | undefined;
  loading?: boolean;
}) {
  const columns: Column<FileRecord>[] = [
    { key: 'type', header: 'Тип', render: (f) => FILE_TYPE_LABELS[f.type] ?? f.type },
    { key: 'name', header: 'Файл', render: (f) => <span className="break-all text-xs">{f.filename}</span> },
    { key: 'size', header: 'Размер', align: 'right', hideOnMobile: true, render: (f) => <span className="num text-xs">{size(f.size_bytes)}</span> },
    { key: 'status', header: 'Статус', render: (f) => <StatusBadge status={f.status} kind="file" /> },
    { key: 'date', header: 'Загружен', hideOnMobile: true, render: (f) => <span className="num text-xs">{formatDateTime(f.confirmed_at ?? f.created_at)}</span> },
    {
      key: 'open',
      header: '',
      align: 'right',
      render: (f) =>
        f.status === 'confirmed' ? (
          <Button size="sm" variant="ghost" onClick={() => openFile(userId, f.id)}>
            <ExternalLink size={14} /> Открыть
          </Button>
        ) : null,
    },
  ];
  return (
    <Table
      columns={columns}
      rows={files}
      rowKey={(f) => f.id}
      loading={loading}
      empty={<EmptyState icon={<FolderOpen size={36} strokeWidth={1.5} />} text="Документов нет" />}
    />
  );
}

// Превью подтверждённых документов: картинки в лайтбоксе, PDF во вкладке.
export function DocumentsGallery({ userId, files }: { userId: string; files: FileRecord[] | undefined }) {
  const [zoom, setZoom] = useState<{ url: string; title: string } | null>(null);
  const confirmed = (files ?? []).filter((f) => f.status === 'confirmed');
  if (!files) return <Skeleton className="h-32 w-full" />;
  if (confirmed.length === 0) return <EmptyState icon={<FolderOpen size={36} strokeWidth={1.5} />} text="Подтверждённых документов нет" />;
  return (
    <>
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
        {confirmed.map((f) => (
          <Thumb key={f.id} userId={userId} file={f} onZoom={setZoom} />
        ))}
      </div>
      <Modal open={!!zoom} onOpenChange={(v) => !v && setZoom(null)} title={zoom?.title} width="max-w-4xl">
        {zoom && <img src={zoom.url} alt={zoom.title} className="mx-auto max-h-[75vh] rounded-lg object-contain" />}
      </Modal>
    </>
  );
}

function Thumb({
  userId,
  file,
  onZoom,
}: {
  userId: string;
  file: FileRecord;
  onZoom: (z: { url: string; title: string }) => void;
}) {
  const isImage = file.content_type.startsWith('image/');
  const url = useQuery({
    queryKey: ['download-url', userId, file.id],
    queryFn: () => filesApi.downloadUrl(userId, file.id),
    enabled: isImage,
    staleTime: 60_000,
  });
  const title = FILE_TYPE_LABELS[file.type] ?? file.type;
  return (
    <button
      type="button"
      className="group overflow-hidden rounded-lg border bg-input text-left"
      onClick={() => (isImage && url.data ? onZoom({ url: url.data.url, title }) : openFile(userId, file.id))}
    >
      <div className="flex h-28 items-center justify-center overflow-hidden">
        {isImage ? (
          url.data ? (
            <img src={url.data.url} alt={title} className="h-full w-full object-cover transition group-hover:scale-105" />
          ) : (
            <Skeleton className="h-full w-full" />
          )
        ) : (
          <FileText size={36} className="text-muted" strokeWidth={1.5} />
        )}
      </div>
      <div className="px-2 py-1.5 text-xs">
        <div className="font-medium">{title}</div>
        <div className="truncate text-muted">{file.filename}</div>
      </div>
    </button>
  );
}
