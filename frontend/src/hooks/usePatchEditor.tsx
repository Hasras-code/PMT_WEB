import { useCallback, useRef, useState } from 'react';
import toast from 'react-hot-toast';
import { api, errMsg } from '../api/client';
import { EditDialog } from '../components/ui';
import type { EditDialogField } from '../components/ui';
import type { JSX } from 'react';

export interface PatchEditorState {
  title: string;
  url: string;
  fields: EditDialogField[];
}

/** Replaces window.prompt() edit flows: opens an accessible dialog that
 * PATCHes the collected values, toasts, and reloads. */
export function usePatchEditor(
  reload: () => void,
  successMessage = 'Updated',
): {
  open: (title: string, url: string, fields: EditDialogField[]) => void;
  dialog: JSX.Element | null;
} {
  const [editing, setEditing] = useState<PatchEditorState | null>(null);
  const [busy, setBusy] = useState(false);
  const busyRef = useRef(false);

  const open = useCallback((title: string, url: string, fields: EditDialogField[]) => {
    setEditing({ title, url, fields });
  }, []);

  const close = useCallback(() => {
    if (!busyRef.current) setEditing(null);
  }, []);

  const submit = useCallback(
    async (values: Record<string, string>) => {
      if (!editing) return;
      busyRef.current = true;
      setBusy(true);
      try {
        await api.patch(editing.url, values);
        toast.success(successMessage);
        setEditing(null);
        reload();
      } catch (err) {
        toast.error(errMsg(err, 'Update failed'));
      } finally {
        busyRef.current = false;
        setBusy(false);
      }
    },
    [editing, reload, successMessage],
  );

  const dialog = editing ? (
    <EditDialog
      key={`${editing.url}:${editing.title}`}
      title={editing.title}
      fields={editing.fields}
      busy={busy}
      onClose={close}
      onSubmit={submit}
    />
  ) : null;
  return { open, dialog };
}
