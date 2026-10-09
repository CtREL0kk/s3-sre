import { useEffect, useMemo, useRef, useState } from 'react'
import ForceGraph2D from 'react-force-graph-2d'
import { Loader2, X } from 'lucide-react'
import { api } from '../api'
import type { GraphNode, ObjectItem } from '../types'
import PreviewDialog from './PreviewDialog'

interface GNode extends GraphNode {
  x?: number
  y?: number
}
type GLinkKind = 'tree' | 'wiki'
interface GLink {
  source: string
  target: string
  target_name?: string
  kind: GLinkKind
}

function drawNode(node: GNode, ctx: CanvasRenderingContext2D, globalScale: number) {
  const radius = node.type === 'folder' ? 8 : 6
  ctx.beginPath()
  ctx.arc(node.x!, node.y!, radius, 0, 2 * Math.PI)

  const color =
    node.type === 'folder'
      ? 'rgba(251, 191, 36, 0.85)'
      : node.visibility === 'public'
        ? 'rgba(52, 211, 153, 0.85)'
        : 'rgba(56, 189, 248, 0.85)'

  ctx.fillStyle = color
  ctx.fill()
  ctx.strokeStyle = 'rgba(255,255,255,0.12)'
  ctx.lineWidth = 1
  ctx.stroke()

  const fontSize = 12 / globalScale
  ctx.font = `${fontSize}px ui-sans-serif, system-ui, sans-serif`
  ctx.textAlign = 'center'
  ctx.textBaseline = 'middle'
  ctx.fillStyle = 'rgba(226, 232, 240, 0.9)'
  ctx.fillText(node.name, node.x!, node.y! + radius + fontSize * 0.55)
}

export default function GraphDialog({ folderId, onClose }: { folderId: string | null; onClose: () => void }) {
  const [nodes, setNodes] = useState<GNode[]>([])
  const [links, setLinks] = useState<GLink[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [preview, setPreview] = useState<ObjectItem | null>(null)
  const [focused, setFocused] = useState<GNode | null>(null)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError('')
    api
      .getGraph(folderId)
      .then((d) => {
        if (cancelled) return
        setNodes(d.nodes.map((n) => ({ ...n })))
        const wiki: GLink[] = d.edges.flatMap((e) =>
          e.target ? [{ source: e.source, target: e.target, target_name: e.target_name, kind: 'wiki' as const }] : [],
        )
        const tree: GLink[] = d.tree.map((t) => ({ source: t.parent, target: t.child, kind: 'tree' as const }))
        setLinks([...tree, ...wiki])
        setLoading(false)
      })
      .catch((e) => {
        if (!cancelled) {
          setError(e instanceof Error ? e.message : 'Ошибка загрузки графа')
          setLoading(false)
        }
      })
    return () => {
      cancelled = true
    }
  }, [folderId])

  const graphData = useMemo(() => ({ nodes, links }), [nodes, links])

  const graphRef = useRef<any>(null)

  useEffect(() => {
    const g = graphRef.current
    if (!g) return
    try {
      const link = g.d3Force?.('link')
      if (link) g.d3Force('link', link.distance(55).strength(0.8))
      const charge = g.d3Force?.('charge')
      if (charge) g.d3Force('charge', charge.strength(-50))
      const collision = g.d3Force?.('collision')
      if (collision) g.d3Force('collision', collision.radius(12))
    } catch {
    }
  }, [])

  async function onNodeClick(node: GNode) {
    if (node.type === 'folder') {
      setFocused(node)
      return
    }
    try {
      const full = await api.getObject(node.id)
      setPreview(full)
    } catch {
      setError('Не удалось открыть файл')
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex flex-col bg-slate-950/90 backdrop-blur">
      <div className="flex items-center justify-between border-b border-slate-800 px-5 py-3">
        <div>
          <h3 className="text-base font-semibold text-slate-100">Граф связей</h3>
          <p className="text-xs text-slate-500">
            {folderId ? `Связи в текущей папке и её поддереве` : 'Связи по всему хранилищу'} · вики-ссылки [[файл]]
          </p>
        </div>
        <button
          onClick={onClose}
          className="flex h-8 w-8 items-center justify-center rounded-lg text-slate-400 hover:bg-slate-800 hover:text-slate-100"
        >
          <X className="h-4 w-4" />
        </button>
      </div>

      <div className="relative flex-1 overflow-hidden">
        {loading && (
          <div className="absolute inset-0 z-10 flex items-center justify-center gap-2 text-slate-400">
            <Loader2 className="h-5 w-5 animate-spin" /> Строим граф…
          </div>
        )}
        {error && <div className="absolute inset-x-0 top-3 z-10 mx-auto w-fit rounded-lg bg-red-950/60 px-4 py-2 text-sm text-red-300">{error}</div>}

        {!loading && nodes.length === 0 && (
          <div className="absolute inset-0 z-10 grid place-items-center text-slate-500">
            Здесь пока нет связей — загрузите markdown-файлы с [[ссылками]].
          </div>
        )}

        <ForceGraph2D
          ref={graphRef}
          graphData={graphData}
          nodeLabel={(n: GNode) => n.name}
          nodeCanvasObject={drawNode}
          nodeCanvasObjectMode={() => 'replace' as const}
          linkColor={(l: GLink) => (l.kind === 'tree' ? 'rgba(148, 163, 184, 0.28)' : 'rgba(103, 232, 249, 0.6)')}
          linkWidth={(l: GLink) => (l.kind === 'tree' ? 1 : 1.6)}
          linkDirectionalParticles={(l: GLink) => (l.kind === 'wiki' ? 2 : 0)}
          linkDirectionalParticleWidth={2}
          linkDirectionalParticleColor={() => 'rgba(103, 232, 249, 0.8)'}
          onNodeClick={onNodeClick}
          backgroundColor="rgba(0,0,0,0)"
          nodeRelSize={8}
          cooldownTicks={100}
          d3AlphaDecay={0.035}
          d3VelocityDecay={0.42}
          nodeVisibility={(n: GNode) => (focused ? focused.id === n.id || links.some((l) => (l.source as string) === focused.id && l.target === n.id) || links.some((l) => l.source === n.id && (l.target as string) === focused.id) : true)}
        />

        {focused && (
          <div className="absolute bottom-4 left-1/2 z-10 flex -translate-x-1/2 items-center gap-3 rounded-full border border-slate-700 bg-slate-900/90 px-4 py-2 shadow-xl">
            <span className="text-sm text-slate-200">Папка: {focused.name}</span>
            <button
              onClick={() => setFocused(null)}
              className="rounded-full bg-slate-800 px-3 py-1 text-xs text-slate-300 hover:bg-slate-700"
            >
              Показать всё
            </button>
          </div>
        )}
      </div>

      {preview && <PreviewDialog item={preview} onClose={() => setPreview(null)} />}
    </div>
  )
}