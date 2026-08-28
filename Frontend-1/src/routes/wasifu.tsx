import { createFileRoute } from "@tanstack/react-router";
import { useEffect, useRef, useState } from "react";
import { AppShell } from "@/components/AppShell";
import { useAuth } from "@/lib/auth-provider";
import { requireAuth } from "@/lib/role-guards";
import { roleMap, type Jukumu } from "@/api/types";
import { useMembers, useCreateMember } from "@/hooks/use-members";
import { User, Phone, Shield, KeyRound, Check, Palette, Camera, Trash2, MapPin, IdCard, Loader2 } from "lucide-react";
import { initials } from "@/lib/utils";
import { authApi } from "@/api/auth";
import { uploadApi } from "@/api/upload";
import { useQueryClient } from "@tanstack/react-query";

export const Route = createFileRoute("/wasifu")({
  head: () => ({ meta: [{ title: "Wasifu wangu — Money Seeking" }] }),
  beforeLoad: () => {
    requireAuth();
  },
  component: WasifuPage,
});

const palette = ["#10b981", "#0ea5e9", "#f59e0b", "#ef4444", "#8b5cf6", "#ec4899", "#14b8a6", "#6366f1"];

function WasifuPage() {
  const { user } = useAuth();
  const qc = useQueryClient();
  const { data: membersData } = useMembers({ page: 1, limit: 1, user_id: user ? String(user.id) : undefined });
  const createMember = useCreateMember();
  const jukumuLabel: Jukumu = user ? (roleMap[user.role] ?? "Mwanachama") : "Mwanachama";

  // Form state — synced with user data via useEffect
  const [jina, setJina] = useState("");
  const [simu, setSimu] = useState("");
  const [anwani, setAnwani] = useState("");
  const [bio, setBio] = useState("");
  const [color, setColor] = useState(palette[0]);
  const [photo, setPhoto] = useState<string | undefined>(undefined);
  const [saving, setSaving] = useState(false);
  const [saveMsg, setSaveMsg] = useState<{ ok: boolean; text: string } | null>(null);
  const [uploading, setUploading] = useState(false);
  const fileRef = useRef<HTMLInputElement>(null);

  const [oldPwd, setOldPwd] = useState("");
  const [newPwd, setNewPwd] = useState("");
  const [pwdMsg, setPwdMsg] = useState<{ ok: boolean; text: string } | null>(null);
  const [pwdLoading, setPwdLoading] = useState(false);

  // Sync form fields whenever user data changes (from API or cache)
  useEffect(() => {
    if (user) {
      setJina(user.name ?? "");
      setSimu(user.phone ?? "");
      setBio(user.bio ?? "");
      setPhoto(user.avatar_url || undefined);
    }
  }, [user?.name, user?.phone, user?.bio, user?.avatar_url]);

  if (!user) return null;

  const me = membersData?.data?.[0] ?? null;

  const saveProfile = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaveMsg(null);
    setSaving(true);
    try {
      const res = await authApi.updateProfile({
        name: jina.trim() || undefined,
        phone: simu.trim() || undefined,
        bio: bio.trim() || undefined,
        avatar_url: photo ?? undefined,
      });
      // Update auth cache so data persists across refreshes
      if (res.data) {
        qc.setQueryData(["auth", "me"], res.data);
      }
      setSaveMsg({ ok: true, text: "Umehifadhiwa ✓" });
      setTimeout(() => setSaveMsg(null), 3000);
    } catch (e: unknown) {
      setSaveMsg({ ok: false, text: e instanceof Error ? e.message : "Imeshindikana kuhifadhi" });
    } finally {
      setSaving(false);
    }
  };

  const onPickPhoto = async (file: File) => {
    setUploading(true);
    setSaveMsg(null);
    try {
      const res = await uploadApi.avatar(file);
      setPhoto(res.url);
    } catch (e: unknown) {
      setSaveMsg({ ok: false, text: e instanceof Error ? e.message : "Imeshindikana kupakia picha" });
    } finally {
      setUploading(false);
    }
  };

  const removePhoto = () => setPhoto(undefined);

  const savePwd = async (e: React.FormEvent) => {
    e.preventDefault();
    setPwdLoading(true);
    setPwdMsg(null);
    try {
      await authApi.changePassword({ old_password: oldPwd, new_password: newPwd });
      setOldPwd(""); setNewPwd("");
      setPwdMsg({ ok: true, text: "Nenosiri limebadilishwa." });
      setTimeout(() => setPwdMsg(null), 2500);
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : "Imeshindikana kubadilisha nenosiri.";
      setPwdMsg({ ok: false, text: msg });
    } finally {
      setPwdLoading(false);
    }
  };

  const becomeMember = async () => {
    try {
      await createMember.mutateAsync({
        full_name: jina || user.name,
        phone: simu || "0700000000",
        address: anwani || undefined,
        joined_at: new Date().toISOString().slice(0, 10),
      });
    } catch { /* handled by RQ */ }
  };


  return (
    <AppShell title="Wasifu wangu" subtitle="Hariri taarifa zako za akaunti">
      <div className="grid gap-5 lg:grid-cols-3">
        <section className="card-surface p-6 lg:col-span-1">
          <div className="flex flex-col items-center text-center">
            <div className="relative">
              {photo ? (
                <img src={photo} alt="" className="h-24 w-24 rounded-full object-cover" />
              ) : (
                <span className="grid h-24 w-24 place-items-center rounded-full text-2xl font-bold text-white" style={{ background: color }}>
                  {initials(jina || user.name)}
                </span>
              )}
              <button
                type="button"
                onClick={() => fileRef.current?.click()}
                disabled={uploading}
                className="absolute -bottom-1 -right-1 grid h-9 w-9 place-items-center rounded-full bg-primary text-primary-foreground shadow-md ring-2 ring-card disabled:opacity-60"
                aria-label="Badilisha picha"
              >
                {uploading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Camera className="h-4 w-4" />}
              </button>
              <input
                ref={fileRef}
                type="file"
                accept="image/*"
                className="hidden"
                onChange={(e) => { const f = e.target.files?.[0]; if (f) onPickPhoto(f); e.target.value = ""; }}
              />
            </div>
            {photo && (
              <button onClick={removePhoto} type="button" className="mt-2 inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-destructive">
                <Trash2 className="h-3 w-3" /> Ondoa picha
              </button>
            )}
            <p className="mt-3 font-display text-lg font-semibold">{user.name}</p>
            <p className="text-sm text-muted-foreground">{user.phone}</p>
            <span className="chip mt-2">{jukumuLabel}</span>
            {jukumuLabel === "Mwanachama" && (
              me ? (
                <p className="mt-2 text-xs text-success">Mwanachama #{me.member_no}</p>
              ) : (
                <button onClick={becomeMember} disabled={createMember.isPending} className="mt-3 inline-flex items-center gap-1.5 rounded-lg bg-primary/10 px-3 py-1.5 text-xs font-semibold text-primary disabled:opacity-50">
                  <IdCard className="h-3.5 w-3.5" /> Jisajili kama mwanachama
                </button>
              )
            )}
          </div>
        </section>

        <section className="card-surface p-6 lg:col-span-2">
          <h2 className="font-display text-base font-semibold">Hariri taarifa</h2>
          <form onSubmit={saveProfile} className="mt-4 space-y-3">
            <Inp icon={User} label="Jina kamili" value={jina} onChange={setJina} />
            <Inp icon={Phone} label="Namba ya simu" value={simu} onChange={setSimu} type="tel" />
            <Inp icon={MapPin} label="Anwani / Mji" value={anwani} onChange={setAnwani} />
            <label className="block">
              <span className="mb-1 block text-xs font-medium text-muted-foreground">Maelezo mafupi (bio)</span>
              <textarea value={bio} onChange={(e) => setBio(e.target.value)} rows={2} className="w-full rounded-xl border border-input bg-background px-3 py-2.5 text-sm outline-none focus:border-primary" />
            </label>
            <label className="block">
              <span className="mb-1 block text-xs font-medium text-muted-foreground">Jukumu</span>
              <div className="flex items-center gap-2 rounded-xl border border-input bg-background px-3 py-2.5">
                <Shield className="h-4 w-4 text-muted-foreground" />
                <span className="text-sm">{jukumuLabel}</span>
              </div>
            </label>
            <div>
              <span className="mb-2 flex items-center gap-1.5 text-xs font-medium text-muted-foreground"><Palette className="h-3.5 w-3.5" /> Rangi ya avatar</span>
              <div className="flex flex-wrap gap-2">
                {palette.map((c) => (
                  <button
                    type="button"
                    key={c}
                    onClick={() => setColor(c)}
                    className={`h-8 w-8 rounded-full ring-2 ring-offset-2 ring-offset-background ${color === c ? "ring-foreground" : "ring-transparent"}`}
                    style={{ background: c }}
                    aria-label={c}
                  />
                ))}
              </div>
            </div>
            {saveMsg && (
              <p className={`text-sm ${saveMsg.ok ? "text-success" : "text-destructive"}`}>{saveMsg.text}</p>
            )}
            <div className="flex items-center gap-3 pt-2">
              <button type="submit" disabled={saving} className="inline-flex items-center gap-2 rounded-xl bg-primary px-5 py-2.5 text-sm font-semibold text-primary-foreground disabled:opacity-60">
                {saving && <Loader2 className="h-4 w-4 animate-spin" />}
                <Check className="h-4 w-4" /> Hifadhi mabadiliko
              </button>
            </div>
          </form>
        </section>

        <section className="card-surface p-6 lg:col-span-3">
          <h2 className="font-display text-base font-semibold">Badilisha nenosiri</h2>
          <form onSubmit={savePwd} className="mt-4 grid gap-3 md:grid-cols-2">
            <Inp icon={KeyRound} label="Nenosiri la zamani" type="password" value={oldPwd} onChange={setOldPwd} />
            <Inp icon={KeyRound} label="Nenosiri jipya (6+)" type="password" value={newPwd} onChange={setNewPwd} />
            <div className="md:col-span-2 flex items-center gap-3">
              <button type="submit" disabled={pwdLoading} className="inline-flex items-center gap-2 rounded-xl bg-primary px-5 py-2.5 text-sm font-semibold text-primary-foreground disabled:opacity-60">
                {pwdLoading ? "Inabadilisha..." : "Badilisha"}
              </button>
              {pwdMsg && (
                <p className={`text-sm ${pwdMsg.ok ? "text-success" : "text-destructive"}`}>{pwdMsg.text}</p>
              )}
            </div>
          </form>
        </section>
      </div>
    </AppShell>
  );
}

function Inp({ icon: Icon, label, value, onChange, type = "text" }: { icon: any; label: string; value: string; onChange: (v: string) => void; type?: string }) {
  return (
    <label className="block">
      <span className="mb-1 block text-xs font-medium text-muted-foreground">{label}</span>
      <div className="flex items-center gap-2 rounded-xl border border-input bg-background px-3 py-2.5 focus-within:border-primary">
        <Icon className="h-4 w-4 text-muted-foreground" />
        <input type={type} value={value} onChange={(e) => onChange(e.target.value)} className="w-full bg-transparent text-sm outline-none" />
      </div>
    </label>
  );
}
