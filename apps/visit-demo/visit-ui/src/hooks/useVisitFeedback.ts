import { useCallback, useState } from "react";

export type Tone = "ok" | "err";

export function useVisitFeedback(initialMessage: string, initialTone: Tone) {
  const [message, setMessage] = useState(initialMessage);
  const [toneClassName, setToneClassName] = useState<Tone>(initialTone);

  const showOk = useCallback((nextMessage: string): void => {
    setMessage(nextMessage);
    setToneClassName("ok");
  }, []);

  const showError = useCallback((nextMessage: string): void => {
    setMessage(nextMessage);
    setToneClassName("err");
  }, []);

  return { message, toneClassName, showOk, showError };
}
