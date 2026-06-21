import { useMemo, useRef } from 'react';
import { useFrame } from '@react-three/fiber';
import * as THREE from 'three';

interface EdgeTubeProps {
  start: [number, number, number];
  end: [number, number, number];
  protocol?: string;
  bandwidth?: number;
}

export default function EdgeTube({ start, end, protocol = 'tcp', bandwidth = 0 }: EdgeTubeProps) {
  const meshRef = useRef<THREE.Mesh>(null);

  const { curve, color } = useMemo(() => {
    const startVec = new THREE.Vector3(...start);
    const endVec = new THREE.Vector3(...end);
    const mid = new THREE.Vector3().lerpVectors(startVec, endVec, 0.5);
    mid.y += 0.5;

    const curve = new THREE.QuadraticBezierCurve3(startVec, mid, endVec);

    const color = protocol === 'tcp' ? '#00d4ff' : '#a855f7';

    return { curve, color };
  }, [start, end, protocol]);

  const tubeGeometry = useMemo(() => {
    const radius = Math.min(0.02 + bandwidth * 0.001, 0.1);
    return new THREE.TubeGeometry(curve, 32, radius, 8, false);
  }, [curve, bandwidth]);

  useFrame(() => {
    if (meshRef.current) {
      const material = meshRef.current.material as THREE.MeshStandardMaterial;
      if (material.emissiveIntensity !== undefined) {
        material.emissiveIntensity = 0.3 + Math.sin(Date.now() * 0.005) * 0.2;
      }
    }
  });

  return (
    <mesh ref={meshRef} geometry={tubeGeometry}>
      <meshStandardMaterial
        color={color}
        metalness={0.3}
        roughness={0.5}
        emissive={color}
        emissiveIntensity={0.3}
        transparent
        opacity={0.8}
      />
    </mesh>
  );
}
