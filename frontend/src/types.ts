export type ObjectType = 'folder' | 'file'
export type Visibility = 'private' | 'public'
export type Permission = 'read' | 'write'

export interface ObjectItem {
  id: string
  parent_id: string | null
  name: string
  type: ObjectType
  visibility: Visibility
  size_bytes: number | null
  content_type: string | null
  etag: string | null
  created_at: string
  updated_at: string
}

export interface Grant {
  grantee_id: string
  permission: Permission
  created_at: string
}

export interface User {
  id: string
  username: string
  email: string
}

export interface GraphNode {
  id: string
  name: string
  type: ObjectType
  visibility: Visibility
  parent_id: string | null
}

export interface GraphEdge {
  source: string
  target: string | null
  target_name: string
}

export interface GraphTreeEdge {
  parent: string
  child: string
}