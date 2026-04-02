import sys
import subprocess

def generate(app, pkg, spec, extPkg):
    print ("generate", app, pkg, spec)
    sbigen = "bin/sbigen"
    subprocess.run([sbigen, "--app", app, "--pkg", pkg, "--spec", spec, "--ext-pkg", extPkg, "--model-ow"], shell=False, text=True)

def main():
    apps = {
            "nsm": {
                "amfman": "ETRI_NSM_Amf_Management.yaml"
                },
            "amf": {
                "comm": "TS29518_Namf_Communication.yaml",
                "handover": "ETRI_AMF_Handover.yaml",
                "uectx": "ETRI_AMF_UeCtx_Management.yaml",
                "n2nas": "ETRI_AMF_N2Nas.yaml",
                "callback": "ETRI_AMF_Callbacks.yaml"
                },
            "smf": {
                "pdu":"TS29502_Nsmf_PDUSession.yaml",
                "n1n2ul": "ETRI_SMF_N1N2.yaml",
                },
            "pran": {
                "n1n2dl": "ETRI_PRAN_N1N2.yaml",
                "uectx": "ETRI_PRAN_UeCtx_Management.yaml",
                "nasdl": "ETRI_PRAN_Nas.yaml",
                "handover": "ETRI_PRAN_Handover.yaml",
                },

            "udm": {
                "sdm":"TS29503_Nudm_SDM.yaml",
                "ueauth":"TS29503_Nudm_UEAU.yaml",
                "uecm":"TS29503_Nudm_UECM.yaml",
                "ueid":"TS29503_Nudm_UEID.yaml"
                },
            "ausf": {
                "ueauth": "TS29509_Nausf_UEAuthentication.yaml"
                },
            "pcf": {
                "smpol":"TS29512_Npcf_SMPolicyControl.yaml",
                "ampol":"TS29507_Npcf_AMPolicyControl.yaml",
                "uepol":"TS29525_Npcf_UEPolicyControl.yaml"
                },
            "udr": {
                "subs":"TS29505_Subscription_Data.yaml",
                "policy":"TS29519_Policy_Data.yaml"
                }
            }

    for app, pkgs in apps.items():
        for pkg, specFile in pkgs.items():
            generate(app, pkg, specFile,"")
   
    generate("upf","n4sbi","ETRI_UPF_N4.yaml","etrib5gc/pfcp/message")
    return 0

if __name__ == '__main__':
    sys.exit(main())
